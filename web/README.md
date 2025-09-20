# SRAKE Web Application - SRA Knowledge Engine Interface

A modern web interface for **SRAKE (SRA Knowledge Engine)**, the comprehensive SRA metadata search and analysis platform.

*SRAKE pronunciation: Like Japanese sake (酒) — "srah-keh"*

## Features

- **Search Interface**: Full-text and semantic search across SRA metadata
- **Browse Collections**: Explore data by organism, platform, or strategy
- **Study Details**: View comprehensive information about studies, experiments, samples, and runs
- **Export Data**: Download search results in various formats (CSV, JSON, TSV)
- **Real-time Statistics**: Dashboard with database statistics and metrics
- **Advanced Filtering**: Filter by library strategy, platform, organism, and more
- **Responsive Design**: Works on desktop and mobile devices

## Technology Stack

- **Frontend**: SvelteKit 2.0 with Svelte 5
- **UI Components**: Custom components with Tailwind CSS
- **Icons**: Lucide icons
- **API**: SRAKE RESTful API built with Go
- **Database**: SQLite with FTS5 for full-text search
- **Search**: SRAKE hybrid search engine with text and vector capabilities

## Development

### Prerequisites

- Node.js 20+
- Go 1.25+
- SQLite 3

### Setup

1. Install dependencies:
```bash
cd web
npm install
```

2. Start the development server:
```bash
npm run dev
```

3. In another terminal, start the API server:
```bash
cd ..
make server
```

4. Open http://localhost:5173 in your browser

### Build for Production

```bash
npm run build
```

The built application will be in the `build` directory.

## Docker Deployment

### Using Docker Compose (Recommended)

```bash
# Start the webapp
make docker-compose-up

# Stop the webapp
make docker-compose-down
```

### Manual Docker Build

```bash
# Build the SRAKE webapp image
docker build -f Dockerfile.webapp -t srake-webapp:latest .

# Run the SRAKE container
docker run -p 8080:8080 -v $(pwd)/data:/data srake-webapp:latest
```

## SRAKE API Endpoints

The SRAKE web application communicates with the following API endpoints:

- `GET /api/v1/search` - Search SRA metadata
- `GET /api/v1/stats` - Get database statistics
- `GET /api/v1/health` - Health check
- `GET /api/v1/studies/{id}` - Get study details
- `GET /api/v1/samples/{id}` - Get sample details
- `GET /api/v1/runs/{id}` - Get run details
- `POST /api/v1/export` - Export search results
- `GET /api/v1/aggregations/{field}` - Get field aggregations

## Project Structure

```
web/
├── src/
│   ├── routes/          # SvelteKit pages
│   │   ├── +page.svelte       # Dashboard
│   │   ├── search/            # Search interface
│   │   ├── browse/            # Browse collections
│   │   ├── export/            # Export data
│   │   └── settings/          # Settings page
│   ├── lib/
│   │   ├── api.ts            # API client
│   │   ├── utils.ts          # Utility functions
│   │   └── components/       # UI components
│   └── app.html              # HTML template
├── static/               # Static assets
├── package.json         # Dependencies
└── vite.config.js      # Vite configuration
```

## Configuration

### Environment Variables

- `PUBLIC_API_URL` - API base URL (default: `/api/v1`)
- `NODE_ENV` - Environment (development/production)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `npm test`
5. Submit a pull request

## License

MIT License - see LICENSE file for details
