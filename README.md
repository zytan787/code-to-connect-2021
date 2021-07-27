# code-to-connect-2021
## FICC - Trade Compression

### 1. Backend
1. Install golang https://golang.org/doc/install.
2. Create a file `.env` in this directory `code-to-connect-2021/backend`.
3. Write this line in the file created above `FRONTEND_HOST=http://localhost:3000`.
4. Open a terminal and cd into `code-to-connect-2021/backend`.
5. Run this command: `go run main.go`.
6. The process should be running and listening to port 8080.

### 2. Frontend
1. Install yarn https://classic.yarnpkg.com/en/docs/install.
2. Create a file `.env.local` in this directory `code-to-connect-2021/frontend`.
3. Write this line in the file created above `BACKEND_HOST=http://localhost:8080`.
4. Open a terminal and cd into `code-to-connect-2021/frontend`.
5. Run this command: `yarn build`.
6. After it is done, run `yarn start`.
7. The process should be running and listening to port 3000.