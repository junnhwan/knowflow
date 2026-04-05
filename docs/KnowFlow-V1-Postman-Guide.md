# KnowFlow V1 Postman 测试指南

## 文件位置

- Collection: [KnowFlow-V1.postman_collection.json](/d:/dev/my_proj/knowflow/docs/postman/KnowFlow-V1.postman_collection.json)
- Environment: [KnowFlow-V1.local.postman_environment.json](/d:/dev/my_proj/knowflow/docs/postman/KnowFlow-V1.local.postman_environment.json)

## 导入方式

1. 打开 Postman。
2. 点击 `Import`。
3. 选择上面两个 JSON 文件一起导入。
4. 右上角切换环境到 `KnowFlow V1 Local`。

## 建议测试顺序

1. `01-健康检查`
2. `02-上传文档-原始文本`
3. `04-普通问答`
4. `06-查询会话列表`
5. `07-查询会话消息`
6. `08-知识反写`
7. `09-重建索引`
8. `10-查看指标`

## 关于 SSE

- `05-流式问答-SSE` 已经放进 Collection。
- 但 Postman 对 SSE 的调试体验一般，能发请求，不一定适合观察完整流。
- 真正看流式效果，更推荐：

```bash
curl -N -X POST "http://localhost:8080/api/chat/query/stream" ^
  -H "Content-Type: application/json" ^
  -H "X-User-ID: demo-user" ^
  -d "{\"session_id\":\"你的session_id\",\"message\":\"继续解释一下 Redis 记忆压缩的设计\"}"
```

## 变量说明

- `base_url`: 服务地址
- `user_id`: 请求头里的 `X-User-ID`
- `session_id`: 由普通问答接口自动回填
- `document_id`: 由上传文档接口自动回填
- `raw_content`: 默认测试文档内容

## 当前注意点

- `02-上传文档-原始文本` 会自动保存 `DocumentID` 到 `document_id`
- `04-普通问答` 会自动保存 `session_id`
- `03-上传文档-文件模式` 需要你在 Postman 里手动选择本地文件
