// Package uploads 提供店铺 logo 等图片的上传/删除接口
//
// 存储策略：
//   - 文件落在 `./uploads/logos/<tenant_id>.<ext>`（相对 Go 进程工作目录，容器内即 /app/uploads/...）
//   - URL 存到 tenant_settings.store_logo_url，格式 `/uploads/logos/<tenant_id>.<ext>?t=<unix>`
//   - 前端拼 apiBase 后使用；后端通过 StaticFS 暴露 `/uploads/*`
//   - 每次上传先删旧文件，避免残留
package uploads

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// UploadRoot 静态文件根目录；main.go 的 e.Static("/uploads", UploadRoot) 复用此常量
const UploadRoot = "./uploads"

const (
	logoSubdir     = "logos"
	maxLogoBytes   = 2 * 1024 * 1024 // 2MB
	logoSettingKey = "store_logo_url"
)

var allowedExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true,
}

// allowedMIME 与 allowedExt 对应：net/http.DetectContentType 识别到的类型必须在白名单内
var allowedMIME = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

// UploadLogo POST /api/store/logo (multipart file=<file>)
func UploadLogo(c *echo.Context) error {
	t := mw.TenantFrom(c)

	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "缺少文件字段 file")
	}
	if file.Size > maxLogoBytes {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("文件大小超出 %d MB 限制", maxLogoBytes/1024/1024))
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExt[ext] {
		return echo.NewHTTPError(http.StatusBadRequest, "仅支持 png / jpg / webp")
	}

	// Magic bytes 校验：防止扩展名伪造
	src, err := file.Open()
	if err != nil {
		return mw.InternalError(c, "open: ", err)
	}
	defer src.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	head = head[:n]
	ct := http.DetectContentType(head)
	if !allowedMIME[ct] {
		return echo.NewHTTPError(http.StatusBadRequest, "文件内容与扩展名不匹配（疑似非图片）")
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return mw.InternalError(c, "seek: ", err)
	}

	// 确保目录
	dir := filepath.Join(UploadRoot, logoSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return mw.InternalError(c, "mkdir: ", err)
	}

	// 清理同一 tenant 的所有旧 logo（不再用 tenant_id 前缀；扫 tenant_prefix_*）
	_ = removeOldLogos(dir, t.ID.String())

	// 目标扩展名：PNG/JPEG 统一重编码（剥离 polyglot payload），WebP 保留原字节
	targetExt := ext
	if ext == ".jpg" || ext == ".jpeg" {
		targetExt = ".jpg"
	}
	// 文件名加 8 字节随机后缀：防路径直接用 tenant_id 猜测暴露租户列表
	// 格式：<tenant_id>_<rand16hex><ext>；tenant_id 仍存在以便 removeOldLogos 清理
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return mw.InternalError(c, "rand: ", err)
	}
	fileName := t.ID.String() + "_" + hex.EncodeToString(suffix) + targetExt
	outPath := filepath.Join(dir, fileName)
	tmpPath := outPath + ".tmp"

	out, err := os.Create(tmpPath)
	if err != nil {
		return mw.InternalError(c, "create: ", err)
	}
	// 根据类型做重编码或原样写入
	writeErr := writeSanitized(out, src, ct)
	closeErr := out.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return echo.NewHTTPError(http.StatusInternalServerError, "sanitize: "+writeErr.Error())
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return echo.NewHTTPError(http.StatusInternalServerError, "close: "+closeErr.Error())
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return mw.InternalError(c, "rename: ", err)
	}

	// 写回 tenant_settings（带 unix 时间戳强制刷新缓存）
	urlPath := fmt.Sprintf("/uploads/%s/%s?t=%d", logoSubdir, fileName, time.Now().Unix())
	q := sqlc.New(mw.TxFrom(c))
	raw, _ := json.Marshal(urlPath)
	if _, err := q.UpsertTenantSetting(c.Request().Context(), sqlc.UpsertTenantSettingParams{
		TenantID: t.ID, Key: logoSettingKey, Value: raw,
	}); err != nil {
		return mw.InternalError(c, "save setting: ", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"url": urlPath})
}

// DeleteLogo DELETE /api/store/logo
func DeleteLogo(c *echo.Context) error {
	t := mw.TenantFrom(c)

	dir := filepath.Join(UploadRoot, logoSubdir)
	_ = removeOldLogos(dir, t.ID.String())

	q := sqlc.New(mw.TxFrom(c))
	if err := q.DeleteTenantSetting(c.Request().Context(), sqlc.DeleteTenantSettingParams{
		TenantID: t.ID, Key: logoSettingKey,
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return mw.InternalError(c, "delete setting: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// writeSanitized 把图片数据写到 out，剥离 polyglot payload
//
// 策略：
//   - PNG / JPEG：用标准库 image.Decode 解出像素 → image/png 或 image/jpeg 编码回去
//     Decode 过程不保留非图像 chunk/APP segment，附加在尾部的恶意字节会被丢弃
//   - WebP：标准库不支持 encode；保留原字节（前面已有 magic bytes + nosniff 响应头兜底）
func writeSanitized(out io.Writer, src io.Reader, mime string) error {
	switch mime {
	case "image/png":
		img, err := png.Decode(src)
		if err != nil {
			return fmt.Errorf("decode png: %w", err)
		}
		return png.Encode(out, img)
	case "image/jpeg":
		img, _, err := image.Decode(src)
		if err != nil {
			return fmt.Errorf("decode jpeg: %w", err)
		}
		return jpeg.Encode(out, img, &jpeg.Options{Quality: 85})
	case "image/webp":
		// 暂不重编码（标准库无 WebP encoder）；nosniff + CSP 响应头兜底
		if _, err := io.Copy(out, src); err != nil {
			return fmt.Errorf("copy webp: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported mime: %s", mime)
	}
}

// removeOldLogos 删除以 <tenantID>. 开头的所有文件
func removeOldLogos(dir, tenantID string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	// 匹配新格式 <tenant_id>_xxx.ext 和老格式 <tenant_id>.ext（兼容过渡）
	newPrefix := tenantID + "_"
	oldPrefix := tenantID + "."
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, newPrefix) || strings.HasPrefix(name, oldPrefix) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}
