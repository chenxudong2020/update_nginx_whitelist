# update_nginx_whitelist
自动更新nginx白名单程序,把cloudflare和gcore cdn的ip段加入nginx白名单，每天凌晨3点更新allow.conf然后reload nginx


安装nginx之后配置 include /etc/nginx/allow.conf
