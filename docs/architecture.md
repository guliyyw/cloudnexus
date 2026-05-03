# CloudNexus 架构设计

详见开发文档 v3.0。

## 核心原则

- 所有服务无状态，可水平扩容
- 共享 PostgreSQL + Redis + MinIO 作为数据层
- IM 跨节点消息通过 Redis Pub/Sub 路由
- 单机 docker-compose 起步，后期迁移到集群
