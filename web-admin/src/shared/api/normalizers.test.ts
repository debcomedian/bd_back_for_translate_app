import { describe, expect, it } from "vitest";
import {
  asArray,
  pickBoolean,
  pickNullableNumber,
  pickNullableString,
  pickNumber,
  pickString,
} from "./normalizers";

describe("normalizers", () => {
  it("selects first available string field", () => {
    expect(pickString({ name_ru: "Категория", name_en: "Category" }, ["title", "name_ru"])).toBe("Категория");
  });

  it("converts numeric strings to numbers", () => {
    expect(pickNumber({ total: "42" }, ["total"])).toBe(42);
  });

  it("returns fallback for invalid numbers", () => {
    expect(pickNumber({ total: "abc" }, ["total"], 7)).toBe(7);
  });

  it("supports nullable string values", () => {
    expect(pickNullableString({ value: "" }, ["value"])).toBeNull();
    expect(pickNullableString({ value: "A1" }, ["value"])).toBe("A1");
  });

  it("supports nullable numeric values", () => {
    expect(pickNullableNumber({ value: "" }, ["value"])).toBeNull();
    expect(pickNullableNumber({ value: "0.45" }, ["value"])).toBe(0.45);
  });

  it("normalizes boolean-like values", () => {
    expect(pickBoolean({ active: "true" }, ["active"])).toBe(true);
    expect(pickBoolean({ active: 0 }, ["active"], true)).toBe(false);
  });

  it("returns array only for array input", () => {
    expect(asArray([1, 2, 3])).toEqual([1, 2, 3]);
    expect(asArray("not-array")).toEqual([]);
  });
});
