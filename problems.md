## Проблемы/вопросы
#### 1. **Ошибка в спецификации**
В спецификации openapi.yml была неточность в /pullRequest/reassign. В schema требовалось old_user_id: { type: string }, тогда как в example -- old_reviewer_id: u2. Был выбран вариант с old_user_id.

#### 2. **Аутентификация**
В задании не было сказано про необходимость JWT-аутентификации, было решено сделать Static Token Authentication с разделением на Admin и User токены. При необходимости легко расширяется до JWT. 