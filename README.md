# Teleblog

Template to create your own site from Telegram channel.

Demo: [davidshekunts.com](https://davidshekunts.com)

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

# How to use

1. Configuration
    1. Create bot in [@BotFather](t.me/BotFather)
    1. `cd cmd/teleblog && cp app.env.example app.env` and fill it with your data
1. Run
    1. `make serve-teleblog` to run Teleblog + Pocketbase admin panel
    1. Go to `127.0.0.1:8090/_` to see Pocketbase admin panel and fill in your user
    1. Create "verification_token"
    1. Send this token to your bot `/verifytoken YOUR_TOKEN`
    1. Add bot to public TG channels and their groups
    1. Send group links to your bot `/addchannel YOUR_CHANNEL_LINK`
1. Upload history messages
    1. Export JSON history from your channel
    1. Via terminal
        1. Paste it to `cmd/teleblog` folder
        1. Run `cd cmd/teleblog && go run . upload-history YOUR_HISTORY.json` (! DONT FORGET to upload channel posts firstly and linked groups posts afterwards)
    1. Via UI
        1. Run your application
        1. Authorize in admin panel (`/_`)
        1. Go to `/_/upload-history` and upload file to it
1. Customization
    1. Change [base_layout.templ](cmd/teleblog/httpapi/views/base_layout.templ) google tag manager
    1. Change [base_layout.templ](cmd/teleblog/httpapi/views/base_layout.templ) meta tags
    1. Change [index.templ](cmd/teleblog/httpapi/views/index.templ) with your profile information
    1. Change any template as you need
1. Deploy Preparations
    1. Create Digital Ocean droplet
    1. `cp .env.example .env` and fill it
    1. Run `make setup-droplet` (it will configure autorestarts and nginx)
    1. Change ENV in `app.env` in droplet from `LOCAL` to `PRODUCTION`
1. Deploy Manual
    1. Run `make deploy` (it will build and deploy Teleblog to your droplet)
1. Deploy Automatic (Github Actions)
    1. Create new ssh key `ssh-keygen` with custom name (don't use passphrase)
    1. Add public key to your droplet `~/.ssh/authorized_keys`
    1. Create Repository Environment named `prod` (github.com/USER/REPOSITORY/settings/environments)
    1. Set 3 secrets:
        1. SERVER_IP – your droplet IP
        1. SSH_PRIV_KEY – your custom private key
        1. SSH_PUB_KEY – your custom public key
    1. Push to master and check actions

# Roadmap

## First phase

MG: Make it so content appears, but customization through Pocketbase admin

Status: Done

## Second phase

MG: Add content improvement features

1. ~~Search~~
1. ~~Extract tags~~
1. ~~Tags filter~~
1. ~~Quote replied comment~~
1. ~~Images~~
    1. ~~Webhook~~
    1. ~~History~~
    1. ~~UI~~
        1. ~~Index~~
        1. ~~Post page~~
    1. ~~Zoom~~
    1. ~~Multiple~~
        1. ~~Webhook~~
        1. ~~History~~
        1. ~~UI~~
            1. ~~Index~~
            1. ~~Post page~~
1. ~~Videos~~
1. ~~Links preview~~
1. ~~SEO~~
    1. ~~Sitemap & robots.txt~~
    1. ~~Meta title~~
    1. ~~Meta description~~
    1. ~~Meta image~~
1. ~~Remove CSS, JS cache~~
1. ~~Deploy DB backup~~

## Third phase

MG: ...

1. Schema.org + Open Graph.
1. Auto-translate
1. Repost to Medium
1. Files
1. Spoilers for Audio & Circles
1. ...

## X phase

1. Dark / Light theme changer
1. Delete old tags
1. Author Image ([getUserProfilePhotos](https://core.telegram.org/bots/api#getuserprofilephotos))
1. Admin page
1. Partial reload
1. Sorting
1. ...

## Don't work with History Messages

1. Pined messages
1. Likes counter
1. ...