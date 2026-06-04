## Building a self-hosted streaming music application

### Build and run

1. Clone this repository:
```bash
git clone https://github.com/Grivvus/ym
cd ym
```

2. Configure environment files:
```bash
cp .env.example .env # adjust variables if needed
touch .env.minio
```
If you want to enable password reset by email, configure the `PASSWORD_RESET_*` and `SMTP_*` variables in `.env`.

3. Configure Docker volumes.

The development compose file uses bind-mounted local directories for PostgreSQL and MinIO. Create the directories configured in `compose.dev.yml`, or change the `driver_opts.device` paths to directories on your host.

4. Build and run Docker containers:
```bash
docker compose -f compose.dev.yml --profile local-storage up --build
```

5. To stop all containers:
```bash
docker compose -f compose.dev.yml --profile local-storage stop
```


### Prometheus + Grafana setup
You need this section only if you want to collect, store, and visualize metrics.

1. Check `prometheus.yml`.

The default configuration scrapes the service container at `service:8000`.

2. Check Grafana provisioning in `grafana/provisioning` and adjust it if needed.

The default provisioning configures the Prometheus datasource and loads dashboards from `grafana/dashboards`.

3. Start containers with '--profile metrics':
```bash
make run-docker
```

4. Open Grafana at http://localhost:3000.

5. To stop all containers:
```bash
make stop-docker
```
