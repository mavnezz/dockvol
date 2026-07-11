<div align="center">
  <h3>Docker volume backup tool</h3>
  <p>DockVol is a free, open source and self-hosted tool to back up your Docker volumes and bind-mounts. Pick a container, choose which of its mounts to include, and stream encrypted backups to remote storage (S3, SFTP, NAS, etc.) with progress notifications (Slack, Discord, Telegram, etc.).</p>
  
  <!-- Badges -->
  [![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
  <br />
  [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](https://github.com/mavnezz/dockvol)
  [![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](https://github.com/mavnezz/dockvol)
  [![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](https://github.com/mavnezz/dockvol)

  <p>
    <a href="#-features">Features</a> •
    <a href="#-installation">Installation</a> •
    <a href="#-usage">Usage</a> •
    <a href="#-license">License</a> •
    <a href="#-contributing">Contributing</a>
  </p>
</div>

---

## ✨ Features

### 📦 **Container-volume backups**

- **Container-centric**: pick a container and DockVol reads its mounts - named volumes *and* bind-mounts - over the Docker socket, so you tick what to back up instead of hunting for paths
- **Root tar-sidecar**: a short-lived sidecar mounts the sources read-only and streams a gzipped tar, so every uid/gid is readable and permissions, ownership and symlinks are preserved for a faithful restore
- **No agent, no mutation**: single host, talks to the local Docker daemon; source data is mounted read-only and never changed

### 🗄️ **Multiple storage destinations**

- **Local storage**: keep backups on your server
- **Remote storage**: S3, Cloudflare R2, Azure Blob, NAS, FTP, SFTP and Rclone
- **Your control**: all data stays on infrastructure you own

### 🔒 **Encryption & security**

- **AES-256-GCM encryption**: backup files are encrypted before they leave the host
- **Zero-trust storage**: encrypted backups stay useless to attackers, so they are safe to keep in shared storage like S3 or Azure Blob
- **Encrypted secrets**: storage and notifier credentials are encrypted and never exposed, even in logs or error messages

### 📱 **Notifications**

- **Multiple channels**: Email, Telegram, Slack, Discord, Microsoft Teams, webhooks
- **Real-time updates**: success and failure notifications

### 👥 **Suitable for teams**

- **Workspaces**: group storages and notifiers for different projects or teams
- **Access management**: role-based permissions - viewer, member, admin or owner
- **Audit logs**: track system activity and user changes

### 🎨 **UX-Friendly**

- **Designer-polished UI**: clean, intuitive interface crafted with attention to detail
- **Dark & light themes**: choose the look that suits your workflow
- **Mobile adaptive**: check your backups from anywhere on any device

### 🐳 **Self-hosted & secure**

- **Single Docker image**: SQLite-backed, with no external database or cache to run
- **Privacy-first**: all your data stays on your infrastructure
- **Open source**: Apache 2.0 licensed, inspect every line of code

### 📦 Installation

You have two ways to install DockVol:

- Simple Docker run
- Docker Compose setup

---

## 📦 Installation

You have two ways to install DockVol: simple Docker run or Docker Compose setup.

### Option 1: Simple Docker run

The easiest way to run DockVol:

```bash
docker run -d \
  --name dockvol \
  -p 4005:4005 \
  -v ./dockvol-data:/dockvol-data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart unless-stopped \
  dockvol/dockvol:latest
```

This single command will:

- ✅ Start DockVol
- ✅ Store all data in `./dockvol-data` directory
- ✅ Mount the Docker socket so DockVol can read your containers' volumes
- ✅ Automatically restart on system reboot

### Option 2: Docker Compose setup

Create a `docker-compose.yml` file with the following configuration:

```yaml
services:
  dockvol:
    container_name: dockvol
    image: ghcr.io/mavnezz/dockvol:latest
    ports:
      - "4005:4005"
    volumes:
      - ./dockvol-data:/dockvol-data
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "dockvol", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 60s
```

Then run:

```bash
docker compose up -d
```

---

## 🚀 Usage

1. **Access the dashboard**: Navigate to `http://localhost:4005`
2. **Add a storage destination**: Local, S3, Cloudflare R2, Azure Blob, NAS, FTP, SFTP or Rclone
3. **Pick a container**: DockVol lists the container's volumes and bind-mounts
4. **Choose what to back up**: Tick the mounts you want to include
5. **Back up**: DockVol streams the selected mounts as an encrypted tar to your storage
6. **Add notifications** (optional): Configure email, Telegram, Slack, Discord, Teams or webhook notifications

### 🔑 Resetting password

If you need to reset the password, you can use the built-in password reset command:

```bash
docker exec -it dockvol ./main --new-password="YourNewSecurePassword123" --email="admin"
```

Replace `admin` with the actual email address of the user whose password you want to reset.

### 💾 Backuping DockVol itself

After installation, it is also recommended to backup your DockVol itself or, at least, to copy secret key used for encryption (30 seconds is needed). So you are able to restore from your encrypted backups if you lose access to the server with DockVol or it is corrupted.

---

## 📝 License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details

## 🤝 Contributing

Contributions are welcome - open an issue or a pull request.