## building self-hosted streaming music application

### build&run

1. clone this repository
```
git clone https://github.com/Grivvus/ym
cd ym
```

2. configure .env files
```bash
cp .env.example .env # you can change some variables
touch .env.minio
```
If you want email password reset, configure `PASSWORD_RESET_*` and `SMTP_*` variables in `.env`.

3. configure docker volumes

4. build and run docker containers
```bash
docker compose -f compose.dev.yml up --build
```
