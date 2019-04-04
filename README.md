# Installation
```bash
cp .env.dist .env
docker build -t anboo/golang-vk-proxy .
docker run  -p 8888:8000 --env-file .env anboo/golang-vk-proxy
```

# Usage

POST /requests
```
{
	"requests": [
		{
			"id": "266283c3-caf0-47ac-baf0-a4a827edb77f",
			"method": "users.get",
			"parameters": {
				"user_ids": "31292206,31292206"
			}
		},
		{
			"id": "93951060-4fdb-4c66-ad1f-7906c2c87bac",
			"method": "users.get",
			"parameters": {
				"user_ids": "31292206,31292206"
			}
		},
		{
			"id": "a6d40a91-3d4d-4f21-92a4-7f3c3b705c9c",
			"method": "friends.get",
			"parameters": {
				"user_id": "31292206"
			}
		},
		{
			"id": "e3fe1993-afd3-4f7c-8942-3c14eb4d37dc",
			"method": "friends.get",
			"parameters": {
				"user_id": "267991553"
			}
		}
	]
}
```
