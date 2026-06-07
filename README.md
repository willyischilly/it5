# Веб-сервис для автоматизации планирования работ по развертыванию новых ИС
## Инструкция по запуску
### 1. Клонировать репозиторий

```bash
git clone https://github.com/willyischilly/it5.git
cd it5
```

### 2. Создать файл с переменными окружения

```bash
copy .env.example .env
```

Открыть `.env` и заполнить значения:

```env
DB_USER=postgres
DB_PASSWORD=102030
DB_NAME=planner

JWT_SECRET=your_super_secret_key_min_32_characters

SEED_ADMIN_EMAIL=admin@planner.local
SEED_ADMIN_PASSWORD=admin123456
```

### 3. Запустить проект

```bash
docker compose up -d
```

### 4. Открыть в браузере

```bash
http://localhost
```

### 5. Войти в систему

После первого запуска автоматически создаётся администратор
с данными указанными в `.env`
