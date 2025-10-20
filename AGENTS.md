# Orca

Orca is a web application that runs on a desktop and provides a web UI for SST OpenCode. This web UI is mainly targeting mobile devices.

Write modern Go and TypeScript code and adhere to best practices.

Run the formater `just fmt` when you are done.

Write comments that explain "why" not "what" when it's not obvious.

## Backend

The backend provides a Web API for the frontend and is not available for direct user access.

## Frontend

React is used for the frontend with Vite as the bundler.

Use React Query `@tanstack/react-query` where it makes sense.

### Styling

Tailwind CSS v4 is used for styling. No `tailwind.config.js`.
