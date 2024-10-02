# Deploy

## Minichat

```bash
sed "s|__EXEC_START__|path|g" minichat.service > /etc/systemd/system/minichat.service
```

## Webhook

```bash
sed "s|__EXEC_START__|path|g" webhook.service > /etc/systemd/system/webhook.service
```

## Setup

Reload systemd.

```bash
systemctl daemon-reload
```

Start service.

```bash
systemctl start name.service
```

Get status.

```bash
systemctl status name.service
```

Stop service.

```bash
systemctl stop name.service
```

Check logs.

```bash
journalctl -u name.service -f
```
