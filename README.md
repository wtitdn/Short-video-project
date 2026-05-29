# 短视频流项目 <a>线上地址</a>
## 内容fork自布洛克琴 <a href='https://github.com/LeoninCS/feedsystem_video_go'>原项目地址→</a>
## 原项目不足：
- 项目目录结构耦合重，多人开发难以同步。
- 视频文件存储与数据库改动在同一台服务器上，在2H2G的服务器上超过三个的并发请求容易卡死。
- 登录注册界面缺少验证码限制，易被爬虫攻击
- 视频内容应该通过审核发布，需要后台管理
- 评论区可以提供AI总结功能
## 该项目拟定改进点：

- [x] 结构重构为简洁架构

- [x] 使用minio中间件做分布式存储减轻服务器压力

- [ ] 验证码限制

- [ ] AI接入总结
- [ ] 后台管理接入

## 项目结构

```text
renew_video/
├── cmd/                  # 程序启动入口
│   ├── app/              # 后端 API 服务入口
│   ├── worker/           # 后台 Worker 服务入口
│   └── miniotest/        # MinIO 测试入口
│
├── config/               # 项目配置文件
│
├── internal/             # 项目核心业务代码
│   ├── config/           # 配置加载与解析
│   ├── controller/       # 接口控制层，负责接收 HTTP 请求并返回响应
│   ├── db/               # 数据库连接与初始化
│   ├── entity/           # 数据实体定义（含DTO层请求响应），对应数据库表结构
│   ├── middleware/       # 中间件，包括 JWT 鉴权、RabbitMQ 消息处理等
│   ├── repo/             # 数据访问层，封装数据库、MinIO 等存储操作
│   ├── usecase/          # 业务逻辑层，处理视频、用户、点赞、评论等核心业务
│   └── worker/           # 后台任务与通知服务，如 SSE 通知、异步任务处理
│
├── pkg/                  # 可复用的基础组件
│   ├── jwt/              # JWT 相关工具
│   ├── minio/            # MinIO 对象存储封装
│   ├── observability/    # 性能监控与 pprof
│   ├── rabbitmq/         # RabbitMQ 连接与死信队列封装
│   ├── ratelimit/        # 接口限流组件
│   └── redis/            # Redis 缓存与 ZSet 操作封装
│
├── go.mod                # Go 模块依赖
├── go.sum                # Go 依赖锁定文件
└── README.md             # 项目说明文档
```

## 后端部署

本地打包dist,分别在cmd/app，cmd/worker下执行

```
set GOOS=linux    
set GOARCH=amd64  
set CGO_ENABLED=0 
go build main.go
```
在云端服务器上的/srv文件夹下，新建shortvideo文件夹，并按照如图所示构建文件夹
```
shortvideo/
└── deploy/
    ├── Untitled/
    │
    ├── api/
    │   └── dist/
    │       ├── app         
    │       └── worker       
    │
    └── backend/
        └── Dockerfile       
```
上传到云端使用dockerfile进行docker镜像构建
```
sudo docker build -f deploy/backend/Dockerfile -t video-api:latest .
```
网络初始化
```
docker network create video-net
```
启动镜像
```
sudo docker run -d --name video-api \
  --network video-net \
  -p 7878:8080 --restart=always \
  -v /etc/shortvideo/config.yaml:/app/config.yaml:ro \
  -e CONFIG_PATH=/app/config.yaml \
  video-api:latest
```
如果产生报错，请通过以下命令自行排查，如果有问题请提交issue
```
docker logs video-api
```