**音屿-Apple TV音乐播放器安装助手**

由 **6uos** 维护，基于 [ATVLoadly](https://github.com/bitxeno/atvloadly) 修改，遵循 **AGPL-3.0**。本仓库是安装助手的公开源码；音屿商业播放器和授权后端独立维护。

- 当前交付：一个 `platform=all` 的轻量 FPK，x86/ARM 共用，由 DockerHub 多架构镜像自动匹配。
- 镜像：[666uos/songlib-tv-installer:0.1.3](https://hub.docker.com/r/666uos/songlib-tv-installer/tags)。
- 默认中文，管理密码8–64位；使用既有音屿圆角图标，提供256/512素材。
- [构建与运行](docs/OPERATIONS.md) · [当前版本说明](docs/RELEASE-0.1.3.md) · [安全说明](docs/SECURITY.md) · [第三方声明](THIRD_PARTY_NOTICES.md)。

本地两架构启动及登录已验证；真实 fnOS、Apple TV 配对/签装与长期续签仍需设备验收。源码已提交不代表已完成上架或生产部署。

下方保留上游 README 作为来源和功能参考；当前衍生版的安装、鉴权及打包说明以上述中文文档为准。

---

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="../preview/logo_dark.svg">
    <img width="500" src="../preview/logo.svg" alt="atvloadly">
  </picture>
</p>


<div align="center">

[![platform](https://img.shields.io/badge/platform-linux%20%7C%20openwrt%20%7C%20macos%20%7C%20windows-989898)](https://github.com/bitxeno/atvloadly/releases)
[![release](https://img.shields.io/docker/v/bitxeno/atvloadly?label=docker%20latest&sort=semver)](https://hub.docker.com/r/bitxeno/atvloadly)
[![Docker Image Size](https://img.shields.io/docker/image-size/bitxeno/atvloadly)](https://hub.docker.com/r/bitxeno/atvloadly)
[![Docker Pulls](https://img.shields.io/docker/pulls/bitxeno/atvloadly)](https://hub.docker.com/r/bitxeno/atvloadly)
[![license](https://img.shields.io/github/license/bitxeno/atvloadly)](https://github.com/bitxeno/atvloadly/blob/master/LICENSE)

</div>

<div align="center">

English | [Svenska](../readme/README_sv.md) | [中文](../readme/README_cn.md) | [Српски (ћирилица)](../readme/README_sr.md) | [Srpski (latinica)](../readme/README_sr_Latn.md)

</div>

atvloadly is a web service that supports sideloading app on Apple TV. It uses [Impactor](https://github.com/claration/Impactor) as the underlying technology for sideloading and automatically refreshes the app to ensure its long-term availability. It runs as a Docker service on Linux/OpenWrt or as a native desktop app on macOS and Windows.

## Features

* Docker running (Linux/OpenWrt) or a native desktop app (macOS / Windows)
* Supports AppleTV pairing
* Supports automatic app refresh
* Supports tracking GitHub releases / AltStore sources, with manual or automatic app updates
* Supports use of multiple Apple ID accounts
* Supports signing with your own certificate (P12 + provisioning profile), without an Apple ID
* I18n support
* Supports installing an app under a custom bundle identifier (Apple ID or P12), next to the original app

## Screenshots

<p align="center">
  <img width="600" src="./docs/preview/home_en.png">
</p>
<p align="center">
  <img width="600" src="./docs/preview/install_en.png">
</p>

## Installation

### Docker (Linux/OpenWrt)

1. The Linux/OpenWrt host needs to install `avahi-daemon`.
   
   **OpenWrt：**
   ```
   opkg install avahi-dbus-daemon
   /etc/init.d/avahi-daemon start
   ```
   
   **Ubuntu:**
   ```
   sudo apt-get -y install avahi-daemon
   sudo systemctl restart avahi-daemon
   ```

2. Please refer to the following command for installation, remember to modify the mount directory.
   
   **Docker:**
   ```
   docker run --security-opt seccomp:unconfined -d --name=atvloadly --restart=always -p 5533:80 -v /path/to/mount/dir:/data -v /var/run/dbus:/var/run/dbus -v /var/run/avahi-daemon:/var/run/avahi-daemon bitxeno/atvloadly:latest
   ```

   The `/var/run/dbus` and `/var/run/avahi-daemon` of the host machine need to be shared with the docker container for use.

   If you want to use the HOST network and want to modify the listening port, you can add environment variables to container:

   ```
   SERVICE_PORT=5533
   ```

   **Docker Compose:**
   ```
   wget https://raw.githubusercontent.com/bitxeno/atvloadly/refs/heads/master/docker-compose.yml
   docker compose pull
   docker compose up -d
   ```

### Desktop app (macOS / Windows)

**macOS 15+ (recommended, Homebrew):**

```
brew install --cask bitxeno/tap/atvloadly
```

**Windows 10/11 (x64 / arm64):**

Download and run the installer (`.exe`) from [Releases](https://github.com/bitxeno/atvloadly/releases).



## Getting Started

### Preparation (very important‼️)

1. A burned account
> Dedicated Apple ID installation account, both free or developer accounts are acceptable (**For security reasons, avoid using commonly used accounts. Instead, create a burned account for installation!**)
> A free Apple ID allows only a few signing certificates, shared by every sideloading tool that uses the account. When atvloadly cannot create one because the account is full, it never revokes a certificate on its own: the install page asks which certificate, if any, may be revoked.
2. A phone to 2FA Verification
> atvloadly needs to be authorized as a trusted device (it will be virtualized as a MacBook). When logging in, Apple will send a 2FA verification code to the registered phone number of your account or to a device that has already logged in with the installation account. Please authorize and verify promptly.

### Operation process

1. Open the Apple TV settings menu, select `Remote and Devices -> Remote App and Devices`, enter pairing mode.
2. Open the web management page, normally it will display the pairable `AppleTV`.
3. Click on the `AppleTV` device to enter the pairing page and complete the pairing operation.
4. After successful pairing, return to the home page, where the connected `AppleTV` will be displayed.
5. Click on the connected `AppleTV` to enter the sideload installation page, select the IPA file that needs to be sideloaded, and click `Install`.

### External certificate (P12) signing

If you already own a code signing certificate, you can import its `.p12` (with its password, which may be empty) and a matching `.mobileprovision` profile (a **tvOS** profile listing the Apple TV UDID for an Apple TV), then choose **External certificate** on the install page. No Apple ID is asked for or used. Apps signed this way are not refreshed automatically: they expire at the earlier of the certificate and profile expiries, and reinstalling with the same certificate and profile does not extend that date. After replacing the profile (or with a new identity), reinstall or update the app to apply the new expiry. They can be installed from and track a source (GitHub releases or AltStore), their updates being signed with the same identity.

The private keys are stored encrypted with a deployment key file (`signing.key_file`, default `/data/keys/signing-identity.key`): back it up together with the database. atvloadly has no authentication, so keep it on a trusted network.

[>> External certificate guide](../external-certificate.md)

## FAQ

1. How many apps can be installed with a free account?

> Each free Apple ID can register up to 10 apps and activate up to 3 apps simultaneously. Installing more than 3 will cause previously installed apps to become unavailable.

2. Unable to find AppleTV

> 1. Please turn off the VPN, restart the AppleTV, re-enter pairing mode, make sure **[Tool]** can detect devices of the `_remotepairing-manual-pairing._tcp` type, and pair again.
>
> 2. Try restarting the `avahi-daemon` service.
>
> 3. Try using `privileged: true` in your docker-compose.yml or `--privileged` in your docker run command instead of `--security-opt seccomp:unconfined`.
>
> 4. On the desktop app (macOS / Windows) no avahi-daemon is involved: check that the Apple TV is on the same network and that the local firewall allows the app to use mDNS.

3. Failed to log in to Apple account

> This may have triggered Apple's risk control. Apple has login restrictions for certain regions. Alternatively, try creating a new account.

4. IPA crashes after installation

> If the IPA requires permissions such as CloudKit, only paid developer accounts can sign and enable them. After sideloading with atvloadly, the IPA's `Bundle Identifier` will be modified, and some IPAs may restrict this, causing crashes.

5. Installation failure after system upgrade.

> After upgrading the system, re-pairing is required. Generally, newly released systems are not supported. It is recommended to disable automatic system updates.

6. Can App-specific passwords be used for passwords? Is it more secure this way?

> Currently does not support it.

## API

- `/healthcheck`: Return service health status (200 indicates normal, 503 indicates that an app has expired).

- `/mcp`: MCP service api, streamable http transport, can connect to AI Agent to install or refresh apps.


## How to build

[>> wiki](https://github.com/bitxeno/atvloadly/wiki/How-to-build)

## Credits

[Impactor](https://github.com/claration/Impactor): the sideload core

[idevice](https://github.com/jkcoxson/idevice): libimobiledevice in pure Rust

[usbmuxd2](https://github.com/tihmstar/usbmuxd2): usbmuxd implementation for linux

[frida-core:](https://github.com/frida/frida-core): remote pairing connection reference

## Disclaimer

* This software is only for learning and communication purposes. The author does not assume any legal responsibility for the security risks or losses caused by the use of this software.
* Before using this software, you should understand and bear corresponding risks, including but not limited to account freezing, which are unrelated to this software.

**音屿衍生版（2026-11-02）**

正式名称：**音屿-Apple TV音乐播放器安装助手**。本体永久免费，音屿播放器独立商业授权，不捆绑IPA。

本地开发/打包见 [操作手册](docs/OPERATIONS.md)、[安全说明](docs/SECURITY.md)、[来源](docs/UPSTREAM.md)。
运行界面可下载对应源码，AGPL-3.0适用。当前为本地开发验收，不宣称已上架或完成Apple TV签装/长期续签测试。
