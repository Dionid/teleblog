package httpapi

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
)

type SitemapURL struct {
	Loc        string
	LastMod    time.Time
	ChangeFreq string
	Priority   string
}

func SiteMapAndRobotsPageHandler(e *core.ServeEvent, app core.App) {
	e.Router.GET("/robots.txt", func(c echo.Context) error {
		baseURL := app.Settings().Meta.AppUrl

		txt := fmt.Sprintf(`User-agent: *
Allow: /
Allow: /post/*
Allow: /public/*
Allow: /sitemap.xml

Disallow: /api/*
Disallow: /admin/*
Disallow: /_/*

# Optimize crawling rate
Crawl-delay: 1

# Main sitemap
Sitemap: %s/sitemap.xml

# Host directive for preferred domain version
Host: %s`, baseURL, baseURL)

		return c.String(http.StatusOK, txt)
	})

	// # sitemap.xml
	e.Router.GET("/sitemap.xml", func(c echo.Context) error {
		posts := []teleblog.Post{}
		err := teleblog.PostQuery(app.Dao()).
			OrderBy("created desc").
			All(&posts)

		if err != nil {
			return err
		}

		baseURL := app.Settings().Meta.AppUrl
		urls := []SitemapURL{
			{
				Loc:        baseURL,
				LastMod:    time.Now(),
				ChangeFreq: "daily",
				Priority:   "1.0",
			},
		}

		for _, post := range posts {
			loc := post.Slug

			if loc == "" {
				loc = post.Id
			}

			changeFreq := "yearly"

			if post.Created.Time().AddDate(0, 0, 7).Before(time.Now()) {
				changeFreq = "weekly"
			}

			urls = append(urls, SitemapURL{
				Loc:        baseURL + "/post/" + loc,
				LastMod:    post.Updated.Time(),
				ChangeFreq: changeFreq,
				Priority:   "0.8",
			})
		}

		// Create XML with proper escaping using encoding/xml
		xmlHeader := `<?xml version="1.0" encoding="UTF-8"?>
		<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`
		xmlFooter := `
		</urlset>`

		var xmlContent string
		xmlContent = xmlHeader

		for _, url := range urls {
			// HTML escape any special characters in the URL
			escapedLoc := html.EscapeString(url.Loc)

			xmlContent += fmt.Sprintf(`
			<url>
				<loc>%s</loc>
				<lastmod>%s</lastmod>
				<changefreq>%s</changefreq>
				<priority>%s</priority>
			</url>`,
				escapedLoc,
				url.LastMod.Format("2006-01-02"),
				url.ChangeFreq,
				url.Priority)
		}

		xmlContent += xmlFooter

		c.Response().Header().Set(echo.HeaderContentType, "application/xml")
		return c.String(http.StatusOK, xmlContent)
	})
}
