frontend:
    cd frontend && pnpm install && pnpm run dev

backend:
    cd backend && FRONTEND_DEV_URL=http://localhost:5173 air

fmt:
    cd backend && go fmt
    cd frontend && pnpm install && pnpm run lint:fix

prod:
    cd frontend && pnpm install && pnpm run build
    rm -rf backend/public/assets
    cp -r frontend/dist/* backend/public/
    cd backend && go run main.go

clean:
    rm -rf frontend/dist
    rm -rf backend/public
