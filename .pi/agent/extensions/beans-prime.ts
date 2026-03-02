import * as fs from "node:fs";
import * as path from "node:path";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

type PrimeCache = {
  text: string;
  configDir: string;
  generatedAt: string;
};

function findBeansConfigDir(startCwd: string): string | null {
  let dir = path.resolve(startCwd);
  while (true) {
    const candidate = path.join(dir, ".beans.yml");
    if (fs.existsSync(candidate)) return dir;
    const parent = path.dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

export default function (pi: ExtensionAPI) {
  let cache: PrimeCache | null = null;

  const refreshPrime = async (cwd: string): Promise<{ ok: boolean; reason?: string }> => {
    const configDir = findBeansConfigDir(cwd);
    if (!configDir) {
      cache = null;
      return { ok: false, reason: "no .beans.yml found" };
    }

    const hasBeans = await pi.exec("which", ["beans"]);
    if (hasBeans.code !== 0) {
      cache = null;
      return { ok: false, reason: "beans CLI not found" };
    }

    const prime = await pi.exec("beans", ["prime"], { cwd: configDir, timeout: 20000 });
    if (prime.code !== 0) {
      cache = null;
      return { ok: false, reason: (prime.stderr || prime.stdout || "beans prime failed").trim() };
    }

    const text = (prime.stdout || "").trim();
    if (!text) {
      cache = null;
      return { ok: false, reason: "beans prime returned empty output" };
    }

    cache = {
      text,
      configDir,
      generatedAt: new Date().toISOString(),
    };

    return { ok: true };
  };

  pi.on("session_start", async (_event, ctx) => {
    await refreshPrime(ctx.cwd);
  });

  pi.on("before_agent_start", async (event) => {
    if (!cache) return;
    const injected = [
      "\n\n[beans-prime context]",
      `generated_at=${cache.generatedAt}`,
      `config_dir=${cache.configDir}`,
      cache.text,
      "[/beans-prime context]",
    ].join("\n");

    return {
      systemPrompt: event.systemPrompt + injected,
    };
  });

  pi.registerCommand("beans-refresh", {
    description: "Refresh beans prime context",
    handler: async (_args, ctx) => {
      const result = await refreshPrime(ctx.cwd);
      if (result.ok) ctx.ui.notify("beans-prime: refreshed", "info");
      else ctx.ui.notify(`beans-prime: skipped (${result.reason})`, "warning");
    },
  });

  pi.registerCommand("beans-status", {
    description: "Show beans prime cache status",
    handler: async (_args, ctx) => {
      if (!cache) {
        ctx.ui.notify("beans-prime: inactive (missing beans CLI or .beans.yml)", "info");
        return;
      }
      ctx.ui.notify(`beans-prime: active (${cache.configDir})`, "info");
    },
  });
}
