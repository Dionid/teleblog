package httpapi

import (
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
		return c.String(http.StatusOK, `User-agent: *
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
Sitemap: https://davidshekunts.ru/sitemap.xml

# Host directive for preferred domain version
Host: davidshekunts.ru`)
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

		baseURL := "https://davidshekunts.ru"
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

			urls = append(urls, SitemapURL{
				Loc:        baseURL + "/post/" + loc,
				LastMod:    post.Created.Time(),
				ChangeFreq: "monthly",
				Priority:   "0.8",
			})
		}

		xmlString := `<?xml version="1.0" encoding="UTF-8"?>
		<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

		for _, url := range urls {
			xmlString += `
			<url>
				<loc>` + url.Loc + `</loc>
				<lastmod>` + url.LastMod.Format("2006-01-02") + `</lastmod>
				<changefreq>` + url.ChangeFreq + `</changefreq>
				<priority>` + url.Priority + `</priority>
			</url>`
		}

		xmlString += `
		</urlset>`

		return c.String(200, xmlString)
	})
}
