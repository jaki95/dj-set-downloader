# DJ Set Downloader

This application provides an API to download DJ sets and split them into individual tracks based on a provided Soundcloud url and tracklist.

## Usage

1. **Start the server:**

   ```bash
   docker compose up
   ```

2. **Send a request to the API:**

   **Example using `curl`:**

   ```bash
   curl -X POST http://localhost:8000/api/process \
   -H "Content-Type: application/json" \
   -d '{
     "downloadUrl": "SOUNDCLOUD_URL_OF_THE_SET",
     "tracklist": "[{\"artist\":\"Artist 1\",\"name\":\"Track 1\",\"startTime\":\"00:00\"},{\"artist\":\"Artist 2\",\"name\":\"Track 2\",\"startTime\":\"03:45\"}]"
   }'
   ```

   Replace `SOUNDCLOUD_URL_OF_THE_SET` with the actual URL of the DJ set and update the `tracklist` JSON with the correct track information.
