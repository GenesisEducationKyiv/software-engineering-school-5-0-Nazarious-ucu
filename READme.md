# Weather Subscription API

**Weather Subscription API** — це REST-сервіс для отримання поточних погодних умов у заданому місті та підписки користувачів на email-оновлення погоди з гнучкою частотою.

## Можливості

* Отримання поточної погоди за містом (через WeatherAPI)
* Підписка на email-оновлення погоди (щогодинно або щоденно)
* Підтвердження email-у через посилання
* Відписка за токеном
* Планове надсилання оновлень відповідно до вибраної частоти
* Листи з підтвердженням
* Swagger-документація для API

## Технології

* Go 1.24+
* Gin (HTTP фреймворк)
* SQLite + modernc.org/sqlite
* SMTP (через SendGrid або інші провайдери)
* Goose (міграції бази)
* Swagger + swag CLI
* Docker + Docker Compose

## Структура проєкту

```
.
├── cmd/WeatherSubscriptionAPI    
├── internal/
│   ├── handlers                       
│   ├── services                  
│   ├── repository                
│   └── templates/                   
├── notifier/                          
├── migrations/                        
├── docs/                              
├── api/swagger.yaml                  
├── Dockerfile                         
├── docker-compose.yml                 
└── .env                               
```

## Налаштування

1. Створи `.env` файл у корені:

    ```
    WEATHER_API_KEY=your_weatherapi_key
    SMTP_USER=verified_user
    SMTP_PASS=your_smtp_password
    SMTP_HOST=smtp.sendgrid.net
    SMTP_PORT=587
    SMTP_FROM=verified_sender@example.com
    ```
    
    Потрібно зареєстуватись на 

2. Запуск проєкту:
    
    ```bash
    docker-compose up --build
    ```

3. Перевірка:
* Swagger: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
* Форма підписки: [http://localhost:8080/](http://localhost:8080/)

## Приклади API-запитів

### Отримання погоди
```
GET /api/weather?city=Lviv
```

### Підписка
```
POST /api/subscribe
Content-Type: application/x-www-form-urlencoded

data: email=test@example.com&city=Lviv&frequency=hourly
```

### Підтвердження email-підписки
```
GET /api/confirm/{token}
```

### Відписка від оновлень
```
GET /api/unsubscribe/{token}
```

## Планове надсилання листів

Оновлення прогнозу погоди надсилаються автоматично залежно від обраної частоти (hourly/daily).

---

MIT License © 2025 [Nazar Parnosov](https://github.com/nazarparnosov)
