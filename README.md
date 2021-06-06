# asperitas-clone
This is clone of website https://asperitas.vercel.app/ \
Frontend part was taken from https://github.com/d11z/asperitas
### Database
in directory database
````
docker compose up
```` 

### Start
````
go build -o bin/main cmd/main.go
bin/main
````

### Test
in directory pkg/handlers
````
go test -v -coverprofile="../../test/user_and_post_cover.out"
go tool cover -html="../../test/user_and_post_cover.out" -o "../../test/user_and_post_cover.html"
````

in directory pkg/user_repo
````
go test -v -coverprofile="../../test/user_repo_cover.out"
go tool cover -html="../../test/user_repo_cover.out" -o "../../test/user_repo_cover.html"
````