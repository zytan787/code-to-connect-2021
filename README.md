# code-to-connect-2021
## FICC - Trade Compression

### 1. Backend
1. Install golang https://golang.org/doc/install.
2. Create a file `.env` in the `backend` directory.
3. Write this line in the file created above `FRONTEND_HOST=http://localhost:3000`.
4. Open a terminal and cd into the `backend` directory.
5. Run this command: `go run main.go`.
6. The process should be running and listening to port 8080.

### 2. Frontend
1. Install yarn https://classic.yarnpkg.com/en/docs/install.
2. Create a file `.env.local` in the `frontend` directory.
3. Write this line in the file created above `BACKEND_HOST=http://localhost:8080`.
4. Open a terminal and cd into the `frontend` directory.
5. Run `yarn install`.
6. Run `yarn build`.
7. After it is done, run `yarn start`.
8. The process should be running and listening to port 3000.