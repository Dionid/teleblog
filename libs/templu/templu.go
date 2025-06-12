package templu

import (
	"context"
	"os"
	"strings"
)

func PathWithVersion(ctx context.Context, url string) string {
	if ctxAppVersion, ok := ctx.Value("APP_VERSION").(string); ok && ctxAppVersion != "" {
		return url + "?v=" + ctxAppVersion
	}

	appVersion := os.Getenv("APP_VERSION")
	if appVersion != "" {
		return url + "?v=" + os.Getenv("APP_VERSION")
	}

	return url + "?v=0.0.1"
}

func RemoveNewLines(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", " "), "\n", "")
}

func OrDefaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
