# SMTP端口配置

## 概述
MailChat现在支持在远程邮件投递中配置自定义SMTP端口。这允许您将邮件发送到非标准端口（非25端口）的服务器。

## 配置选项

### Remote Target模块
在`target.remote`块中添加`smtp_port`配置项：

```
target.remote remote_deliver {
    hostname mx.example.com
    smtp_port 8825  # 自定义SMTP端口，默认为25
}
```

### DANE策略模块
如果使用DANE策略，也可以为TLSA记录查询配置相应的端口：

```
mx_auth {
    dane {
        smtp_port 8825  # 用于TLSA记录查询的端口
    }
}

target.remote remote_deliver {
    hostname mx.example.com  
    smtp_port 8825
    mx_auth &mx_auth
}
```

## 完整配置示例

```
# 基本配置
tls off
hostname mx.example.com

# SMTP监听
smtp tcp://127.0.0.1:8825 {
    targets &remote_deliver
}

# MX认证策略（可选）
mx_auth {
    dane {
        smtp_port 8825  # DANE策略使用的端口
    }
}

# 远程目标配置
target.remote remote_deliver {
    hostname mx.example.com
    smtp_port 8825          # 出站连接端口
    mx_auth &mx_auth       # 应用MX认证策略
}
```

## 使用场景

1. **自定义端口邮件服务器**: 连接到使用非标准端口的邮件服务器
2. **内网邮件转发**: 在内网环境中使用自定义端口进行邮件路由
3. **测试环境**: 在开发和测试环境中使用不同端口避免冲突

## 注意事项

- 默认SMTP端口为25，如不配置`smtp_port`将使用默认值
- DANE策略的端口配置需要与实际连接端口保持一致，以确保TLSA记录查询正确
- 确保防火墙允许访问配置的自定义端口

## 向后兼容性

此更新完全向后兼容，现有配置无需修改即可正常工作。