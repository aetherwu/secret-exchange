package config

import "os"

const Version = "1.0"

var (
	DatabaseDebugURI   = env("DB_DEBUG_URI", "localhost:27017")
	DatabaseTestURI    = env("DB_TEST_URI", "")
	DatabaseReleaseURI = env("DB_RELEASE_URI", "")
	DatabaseUser       = env("DB_USER", "")
	DatabasePassword   = env("DB_PASSWORD", "")

	WxAppID     = env("WX_APP_ID", "")
	WxAppSecret = env("WX_APP_SECRET", "")

	QiniuBucket                      = env("QINIU_BUCKET", "gamecard")
	QiniuAccessKey                   = env("QINIU_ACCESS_KEY", "")
	QiniuSecretKey                   = env("QINIU_SECRET_KEY", "")
	QiniuDownload                    = env("QINIU_DOWNLOAD", "")
	QiniuDownloadImageDefaultQuality = env("QINIU_DOWNLOAD_QUALITY", "?imageMogr2/auto-orient/blur/1x0/quality/75|imageslim")
	QiniuCallBackURL                 = env("QINIU_CALLBACK_URL", "")

	DashboardUser     = env("DASHBOARD_USER", "admin")
	DashboardPassword = env("DASHBOARD_PASSWORD", "changeme")
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
