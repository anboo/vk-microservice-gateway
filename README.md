# Introdaction
All requests to the service vk.com will be executed in parallel, depending on how many processors joined the gateway. All processors must be located on different servers or under different ip addresses in order to bypass request blocking once a second. The gateway itself will distribute all necessary requests between handlers through the round-robin algorithm. Each handler will be able to send only 1 message per second, if you have several handlers, then you can already execute 10 requests in 1 second.

If an error occurs in one of the processors, the task is transferred to another processor, and one penalty point is charged to the erroneous processor. If one of the processors has more than 10 errors, it is temporarily disabled.

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
