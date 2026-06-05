# SOLIDserver testdata

Create `config.json` with your SOLIDserver connection details:

```json
{
  "host": "<SOLIDserver-hostname>",
  "serverName": "<dns-server-name>",
  "zoneName": "<zone-name>"
}
```

Optional fields: `port` (default `443`), `viewName`.

## Running tests

Credentials are provided via environment variables:

```bash
export SOLIDSERVER_USERNAME=<username>
export SOLIDSERVER_PASSWORD=<password>

TEST_DNS_SERVER=<dns-server-ip> TEST_ZONE_NAME=example.com. make test
```

- `SOLIDSERVER_USERNAME` / `SOLIDSERVER_PASSWORD` — credentials (required)
- `TEST_DNS_SERVER` — DNS server for propagation checks (port `:53` appended automatically)
- `TEST_ZONE_NAME` — DNS zone to test against (required, must end with `.`)
