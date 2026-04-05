# Go 后端面试笔记

KnowFlow 的主链路是文档摄取、混合检索、Rerank、引用式回答、Redis 会话记忆、知识反写与索引热更新。

混合检索采用 PgVector 语义召回和 pg_trgm 关键词召回，再用 RRF 融合，并在融合结果上做 Rerank。

Redis 会话记忆采用 recent + summary 双层结构，超过阈值后会压缩历史消息。

知识反写不会直接把内容追加到 markdown 文件，而是先落结构化 knowledge_entries，再触发受影响范围的索引重建。
