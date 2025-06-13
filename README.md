# Teleblog

Template to create your own site from Telegram channel.

Demo: [davidshekunts.ru](https://davidshekunts.ru)

# Stack

1. Go
1. Pocketbase
1. Templ
1. Vue
1. Tailwind
1. daisyUI
1. Digital Ocean
1. Github Actions

# Word of caution

This project is NOT about best practices. It's about making product
and do it efficiently. I haven't been working with Vue for a long time,
and this is first time for me to use Pocketbase, Templ.

Don't take this project as a reference for best practices.

# Project structure

1. `cmd/teleblog` - Teleblog platform
1. `infra` - some infrastructure code (nginx, systemctl)
1. `libs` - libraries

# How to

## Install dependencies

1. Install Go & Node.js
1. Run `make setup`
1. Install node deps (`npm install` or `yarn install` or `pnpm install`)

## Develope

1. Run `cd cmd/teleblog && cp app.env.example app.env` and fill it
1. Run `make dev`

## Deploy

1. From sources
    1. Make sure you have Go installed, GOBIN and PATH configured on server
    1. Clone repository `git clone git@github.com:Dionid/teleblog.git`
    1. Install dependencies (live in section above)
    1. Run `make serve`
1. Go install
    1. Make sure you have Go installed, GOBIN and PATH configured on server
    1. Run `go install github.com/Dionid/teleblog/cmd/teleblog@latest` on server
    1. Run `teleblog serve --http=127.0.0.1:8091`

## Configure

1. Create bot in [@BotFather](t.me/BotFather)
1. Go to `SITE_URL:8090/_` to see Pocketbase admin panel and fill in your admin
1. Create user in `user` table (`username` will be enough)
1. Verify in bot to start parsing your channel
    1. Create `tg_verification_token` (be sure that column "verified" is false)
    1. Send this token to your bot `/verifytoken YOUR_TOKEN` (this will add `tg_id`, `tg_user` and `verified` to your user)
    1. Add bot to public TG channels and their groups
    1. Send group links to your bot `/addchannel YOUR_CHANNEL_LINK`

## Customize

1. Fill logo, seo and description data in `config` table
1. Add menu items in `menu_item` table
1. Change any template as you need in `cmd/teleblog/httpapi`
1. Add any public assets to `cmd/teleblog/httpapi/public`

## Upload history messages

1. Export JSON history from your channel and zip it with files
1. Via UI
    1. Go to `/_/upload-history` and upload file to it
1. Via terminal
    1. Paste it to `cmd/teleblog` folder
    1. Run `cd cmd/teleblog && go run . upload-history FILE_NAME.zip`
1. !ATTENTION! Upload channels posts firstly and linked chats comments secondly

# Roadmap

## First phase

Goal: Make it so content appears, but customization through Pocketbase admin

Status: Done

## Second phase

Goal: Add content improvement features

Status: Done

## Third phase

Goal: Separate it from demo project

1. ~~Custom logo, description, footer and menu~~
1. ~~Custom domain~~

## X phase

Goal: Future features

1. Schema.org + Open Graph.
1. Auto-translate
1. Repost to Medium
1. Files
1. Spoilers for Audio & Circles
1. Dark / Light theme changer
1. Delete old tags
1. Author Image ([getUserProfilePhotos](https://core.telegram.org/bots/api#getuserprofilephotos))
1. Admin page
1. Partial reload
1. Sorting
1. Releases
1. Docker image
1. H1 from any text

# Don't work with History Messages

1. Pinned messages
1. Likes counter
