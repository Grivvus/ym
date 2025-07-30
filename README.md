## want to build streaming application for music like spotify etc.

### build&run

```
cp ym_service/.env.exmaple ym_service/.env # set-up file with env variables
docker compose up --build # to run python service, postgresql and minio
```

navigate to `0.0.0.0:8000/docs` to see api endpoints
