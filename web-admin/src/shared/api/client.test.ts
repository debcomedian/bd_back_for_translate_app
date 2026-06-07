import { describe, expect, it } from "vitest";
import { extractApiError } from "./client";

describe("extractApiError", () => {
  it("returns timeout message for aborted request", () => {
    const error = { isAxiosError: true, code: "ECONNABORTED" };
    expect(extractApiError(error)).toContain("дольше отведённого времени");
  });

  it("returns connection message when response is missing", () => {
    const error = { isAxiosError: true };
    expect(extractApiError(error)).toContain("Не удалось подключиться к серверу");
  });

  it("maps backend validation code to readable message", () => {
    const error = {
      isAxiosError: true,
      response: { status: 422, data: { error: "validation_error" } },
    };
    expect(extractApiError(error)).toBe("Данные не прошли проверку");
  });

  it("uses backend russian message when it is provided", () => {
    const error = {
      isAxiosError: true,
      response: { status: 400, data: { message: "Файл не выбран" } },
    };
    expect(extractApiError(error)).toBe("Файл не выбран");
  });

  it("does not expose english technical error text for generic Error", () => {
    expect(extractApiError(new Error("Network Error"))).toBe("Ошибка при выполнении операции");
  });
});
