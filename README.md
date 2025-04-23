# ListenMail

ListenMail 是一个灵活的邮件监听和分发工具，支持多种邮件协议源和自定义处理器。

## 功能特点

- 支持多种邮件协议源：
  - SMTP 服务器（接收邮件）
  - IMAP 客户端（监听邮箱）
  - POP3 客户端（监听邮箱）
- 灵活的处理器系统
  - 基于模式匹配的邮件分发
  - 自定义处理逻辑
- 配置驱动
  - YAML 配置文件
  - 支持多个邮件源
  - 每个源可独立配置

## 安装

```bash
go install github.com/iamlongalong/listenmail/cmd/listenmail@latest
```

## 配置

创建 `config.yaml` 文件：

```yaml
server:
  addr: "0.0.0.0:80"
  username: "admin"
  password: "admin"
save:
  dir: "./data"
sources:
  smtp:
    - name: local_smtp
      enabled: true

      address: ":25"
      domain: "0.0.0.0"
      read_timeout: 10s
      write_timeout: 10s
      max_message_bytes: 10485760  # 10MB
      max_recipients: 50
      allow_insecure_auth: true

  # imap:
  #   - name: gmail_imap
  #     enabled: true
  #     server: "imap.gmail.com:993"
  #     username: "your-email@gmail.com"
  #     password: "your-app-password"
  #     tls: true
  #     check_interval: 30s

  # pop3:
  #   - name: outlook_pop3
  #     enabled: true
  #     server: "outlook.office365.com:995"
  #     username: "your-email@outlook.com"
  #     password: "your-password"
  #     tls: true
  #     check_interval: 30s

  # mailhog:
  #   - name: local_mailhog
  #     enabled: true
  #     api_url: "http://localhost:8025"
  #     check_interval: 5s 
```

## 使用示例

1. 创建自定义处理器：

```go
type MyHandler struct{}

func (h *MyHandler) Handle(mail *message.Entity) error {
    header := mail.Header
    from, _ := header.Text("From")
    subject, _ := header.Text("Subject")
    log.Printf("收到来自 %s 的邮件，主题：%s", from, subject)
    return nil
}

func (h *MyHandler) Match(mail *message.Entity) bool {
    subject, _ := mail.Header.Text("Subject")
    return strings.Contains(subject, "[重要]")
}
```

> pkg/handlers 下提供了多种 condition 和常用的 handler

2. 注册处理器：

```go
disp := dispatcher.New()
disp.AddHandler(&MyHandler{})
```

## 构建和运行

```bash
go run cmd/listenmail
```

## web 页面
启动后，可以在 http://localhost/ 看到，basic auth 在 config.yaml 中配置

## 注意事项

SMTP 服务器可能需要 root 权限才能监听 25 端口

## DNS 设置 (只用于接收)
1. 域名映射到 ip: A domain ip (你的服务器直接 http 访问的域名)
2. 邮箱域名映射到普通域名: MX mail_domain domain (从邮件域名到 http 访问的域名，可以相同)

## ssh 做服务端口映射
目的是只用服务器的 25 端口做转发，实际的代码运行在本地的，便于做 debug 之类的

1. 保证服务器的 25 端口入网开放
2. 保证 sshd 的 `AllowTcpForwarding yes` 和 `GatewayPorts yes` 配置，并 `systemctl restart ssh` 重启
3. 使用 `ssh -R 25:0.0.0.0:25 root@xx.xx.xx.xx` 来把远端的 25 端口导入到本地的 25 端口

## 许可证

MIT License
