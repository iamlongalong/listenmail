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
go get github.com/iamlongalong/listenmail
```

## 配置

创建 `config.yaml` 文件：

```yaml
sources:
  - type: smtp
    name: local_smtp
    enabled: true
    settings:
      address: ":25"
      domain: "localhost"

  - type: imap
    name: gmail_imap
    enabled: true
    settings:
      server: "imap.gmail.com:993"
      username: "your-email@gmail.com"
      password: "your-app-password"

  - type: pop3
    name: outlook_pop3
    enabled: true
    settings:
      server: "outlook.office365.com:995"
      username: "your-email@outlook.com"
      password: "your-password"
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

2. 注册处理器：

```go
disp := dispatcher.New()
disp.AddHandler(&MyHandler{})
```

## 构建和运行

```bash
go build
./listenmail
```

## 注意事项

1. SMTP 服务器可能需要 root 权限才能监听 25 端口
2. 使用 Gmail IMAP 时需要使用应用专用密码
3. 某些邮件服务可能需要特殊的安全设置

## 许可证

MIT License 