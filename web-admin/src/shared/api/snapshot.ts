import { api } from "./client";
import type { SnapshotResponse } from "../../types";
import { asArray } from "./normalizers";
import { normalizeDirection } from "./directions";

export async function fetchSnapshot(): Promise<SnapshotResponse> {
  const { data } = await api.get<SnapshotResponse>("/content/snapshot");
  return {
    snapshot_version: data.snapshot_version ?? null,
    languages: asArray(data.languages),
    categories: asArray(data.categories),
    concepts: asArray(data.concepts),
    forms: asArray(data.forms),
    concept_meta: asArray(data.concept_meta),
    form_meta: asArray(data.form_meta),
    form_synonyms: asArray(data.form_synonyms),
    directions: asArray(data.directions).map(normalizeDirection),
  };
}
