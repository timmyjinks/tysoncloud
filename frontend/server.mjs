import { createServer } from "node:http";
import { Readable } from "node:stream";
import handler from "./dist/server/server.js";

const port = Number(process.env.PORT ?? 3000);

const server = createServer(async (req, res) => {
  try {
    const url = new URL(req.url ?? "/", `http://${req.headers.host ?? "localhost"}`);
    const init = { method: req.method, headers: req.headers };
    if (req.method !== "GET" && req.method !== "HEAD") {
      init.body = Readable.toWeb(req);
      init.duplex = "half";
    }
    const response = await handler.fetch(new Request(url, init));

    const headers = {};
    for (const [k, v] of response.headers.entries()) headers[k] = v;
    const setCookie = response.headers.getSetCookie?.();
    if (setCookie?.length) headers["set-cookie"] = setCookie;
    res.writeHead(response.status, headers);
    if (response.body) {
      for await (const chunk of Readable.fromWeb(response.body)) res.write(chunk);
    }
    res.end();
  } catch (err) {
    console.error(err);
    if (!res.headersSent) res.writeHead(500, { "content-type": "text/plain" });
    res.end("internal server error");
  }
});

server.listen(port, () => console.log(`frontend listening on :${port}`));
