# Go Line Accounting Bot

A simple accounting bot service built with Go, supporting category management and fast expense/income recording for users.

## Features

- Add income or expense categories
- Quick record of transactions
- Monthly summary reports
- LINE Bot integration

## Usage

- Add a category: `新增類別 支出 早餐`
- Quick record: `早餐 150`
- View all categories: `已設定類別`
- Monthly summary: `結算` or `結算 2025年 5月`
- Help: `指令大全`

## Development & Startup

```bash
go mod tidy
go run main.go
```

## Environment Variables

- Configure your database and LINE Bot credentials in `config.yaml` or via environment variables as needed.

## API Endpoints

- `/callback` : LINE webhook endpoint
- `/health`   : Health check endpoint

## License

MIT