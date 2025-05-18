server {
    server_name davidshekunts.com;

    listen 443 ssl; # managed by Certbot
    ssl_certificate /etc/letsencrypt/live/davidshekunts.com/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/davidshekunts.com/privkey.pem; # managed by Certbot
    include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem; # managed by Certbot

    return 301 https://about-me.davidshekunts.com/;
}

server {
    if ($host = davidshekunts.com) {
        return 301 https://$host$request_uri;
    } # managed by Certbot


    server_name davidshekunts.com;
    listen 80;

    return 404; # managed by Certbot
}