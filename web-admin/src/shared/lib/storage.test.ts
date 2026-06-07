import { beforeEach, describe, expect, it } from "vitest";
import { clearAdminToken, getAdminToken, setAdminToken } from "./storage";

class MemoryStorage {
  private values = new Map<string, string>();

  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }

  removeItem(key: string): void {
    this.values.delete(key);
  }

  clear(): void {
    this.values.clear();
  }
}

describe("admin token storage", () => {
  const storage = new MemoryStorage();

  beforeEach(() => {
    storage.clear();
    Object.defineProperty(globalThis, "localStorage", {
      value: storage,
      configurable: true,
    });
  });

  it("stores admin token", () => {
    setAdminToken("token-1");
    expect(getAdminToken()).toBe("token-1");
  });

  it("clears admin token", () => {
    setAdminToken("token-1");
    clearAdminToken();
    expect(getAdminToken()).toBeNull();
  });
});
