# Web Frontend Development Guidelines

## Tech Stack

- **Framework**: React 19 with TypeScript
- **Build Tool**: Vite (using rolldown-vite)
- **Package Manager**: Bun
- **Styling**: Tailwind CSS v4
- **Routing**: TanStack Router
- **State Management**: TanStack Query (React Query)
- **Forms**: TanStack Form
- **UI Components**: shadcn/ui with Radix UI primitives
- **API Client**: Generated via Orval from OpenAPI spec

## Scripts

### Development

```bash
# Start development server (hot reload enabled)
bun run dev

# Alternative using npm
npm run dev
```

The dev server runs at `http://localhost:5173` with hot module replacement.

### Building

```bash
# Type check and build for production
bun run build

# Alternative using npm
npm run build
```

Output is placed in the `dist/` directory.

### Preview Production Build

```bash
# Preview the production build locally
bun run preview
```

### Linting

```bash
# Run ESLint
bun run lint

# Alternative using npm
npm run lint
```

### API Client Generation

```bash
# Generate TypeScript API client from OpenAPI spec
bun run generate-api

# Alternative using npm
npm run generate-api
```

This uses Orval with the configuration in `orval.config.ts` to generate type-safe API hooks and types from the backend OpenAPI specification.

## Installation

```bash
# Install dependencies
bun install

# Alternative using npm
npm install
```

## Docker

```bash
# Build Docker image
docker build -t virsh-sandbox-frontend .

# Run via docker-compose from project root
docker-compose up web
```

## Project Structure

```
web/
├── src/
│   ├── components/     # React components
│   ├── routes/         # TanStack Router routes
│   ├── lib/            # Utility functions
│   └── api/            # Generated API client (via Orval)
├── public/             # Static assets
├── orval.config.ts     # API generation config
├── vite.config.ts      # Vite configuration
├── tsconfig.json       # TypeScript config
├── components.json     # shadcn/ui config
└── package.json
```

## Configuration Files

- `vite.config.ts` - Vite bundler configuration
- `tsconfig.json` - TypeScript compiler options
- `tsconfig.app.json` - App-specific TS config
- `tsconfig.node.json` - Node-specific TS config
- `eslint.config.js` - ESLint configuration
- `orval.config.ts` - Orval API generation config
- `components.json` - shadcn/ui component configuration

## Development Workflow

1. Start the backend services (API at `:8080`)
2. Run `bun run dev` to start the frontend dev server
3. Changes to source files trigger hot module replacement
4. Run `bun run generate-api` after backend API changes to update types

## Adding UI Components

This project uses shadcn/ui. To add new components:

```bash
# Example: add a button component
bunx shadcn@latest add button
```

## Type Checking

TypeScript type checking is performed as part of the build process. For standalone type checking:

```bash
# Run TypeScript compiler in check mode
bunx tsc --noEmit
```
