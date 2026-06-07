import { api } from "./client";
import type {
  DirectionListResponse,
  DirectionSortBy,
  DirectionSortDir,
  TrainingDirection,
} from "../../types";
import {
  pickBoolean,
  pickNullableNumber,
  pickNullableString,
  pickNumber,
  pickString,
} from "./normalizers";

export type FetchDirectionsParams = {
  page?: number;
  page_size?: number;
  q?: string;
  direction_code?: string;
  source_lang_code?: string;
  target_lang_code?: string;
  active?: "true" | "false" | "all";
  category_id?: number;
  unassigned?: "true" | "false";
  sort_by?: DirectionSortBy;
  sort_dir?: DirectionSortDir;
};

const defaultPagination = {
  page: 1,
  page_size: 100,
  total: 0,
  total_pages: 0,
  has_prev: false,
  has_next: false,
};

export function normalizeDirection(item: unknown): TrainingDirection {
  const directionId = pickNumber(item, [
    "direction_id",
    "DirectionID",
    "DirectionId",
    "id",
    "ID",
  ]);
  return {
    direction_id: directionId,
    id: directionId,
    concept_id: pickNumber(item, ["concept_id", "ConceptID", "ConceptId"]),
    direction_code: pickString(item, ["direction_code", "DirectionCode"]),
    source_lang_code: pickString(item, ["source_lang_code", "SourceLangCode"]),
    target_lang_code: pickString(item, ["target_lang_code", "TargetLangCode"]),
    source_form_id: pickNumber(item, [
      "source_form_id",
      "SourceFormID",
      "SourceFormId",
    ]),
    source_value: pickString(item, ["source_value", "SourceValue"]),
    target_form_id: pickNumber(item, [
      "target_form_id",
      "TargetFormID",
      "TargetFormId",
    ]),
    target_value: pickString(item, ["target_value", "TargetValue"]),
    category_id: pickNullableNumber(item, [
      "category_id",
      "CategoryID",
      "CategoryId",
    ]),
    category_slug: pickNullableString(item, ["category_slug", "CategorySlug"]),
    category_name_ru: pickNullableString(item, [
      "category_name_ru",
      "CategoryNameRu",
    ]),
    category_name_en: pickNullableString(item, [
      "category_name_en",
      "CategoryNameEn",
    ]),
    category_name_de: pickNullableString(item, [
      "category_name_de",
      "CategoryNameDe",
    ]),
    cefr_level: pickNullableString(item, [
      "cefr_level",
      "CefrLevel",
      "CEFRLevel",
    ]),
    importance_score: pickNullableNumber(item, [
      "importance_score",
      "ImportanceScore",
    ]),
    concept_base_difficulty: pickNullableNumber(item, [
      "concept_base_difficulty",
      "ConceptBaseDifficulty",
    ]),
    source_form_score: pickNullableNumber(item, [
      "source_form_score",
      "SourceFormScore",
    ]),
    target_form_score: pickNullableNumber(item, [
      "target_form_score",
      "TargetFormScore",
    ]),
    direction_bias: pickNullableNumber(item, [
      "direction_bias",
      "DirectionBias",
    ]),
    synonym_relief: pickNullableNumber(item, [
      "synonym_relief",
      "SynonymRelief",
    ]),
    final_difficulty: pickNullableNumber(item, [
      "final_difficulty",
      "FinalDifficulty",
    ]),
    is_active: pickBoolean(item, ["is_active", "IsActive"], true),
    created_at: pickString(item, ["created_at", "CreatedAt"], ""),
  };
}

function directionItemsFromResponse(data: unknown): unknown[] {
  if (Array.isArray(data)) return data;

  if (data && typeof data === "object") {
    const obj = data as Record<string, unknown>;

    if (Array.isArray(obj.items)) return obj.items;
    if (Array.isArray(obj.directions)) return obj.directions;
    if (Array.isArray(obj.data)) return obj.data;
  }

  return [];
}

function normalizeDirectionListResponse(
  data: unknown,
  fallbackParams?: FetchDirectionsParams,
): DirectionListResponse {
  const items = directionItemsFromResponse(data).map(normalizeDirection);

  if (!data || typeof data !== "object" || Array.isArray(data)) {
    return {
      items,
      total: items.length,
      pagination: {
        page: fallbackParams?.page ?? 1,
        page_size: fallbackParams?.page_size ?? items.length,
        total: items.length,
        total_pages: items.length > 0 ? 1 : 0,
        has_prev: false,
        has_next: false,
      },
      sort: {
        sort_by: fallbackParams?.sort_by ?? "direction_id",
        sort_dir: fallbackParams?.sort_dir ?? "asc",
      },
    };
  }

  const obj = data as Record<string, unknown>;
  const pagination =
    obj.pagination && typeof obj.pagination === "object"
      ? (obj.pagination as Record<string, unknown>)
      : {};
  const sort =
    obj.sort && typeof obj.sort === "object"
      ? (obj.sort as Record<string, unknown>)
      : {};
  const total = Number(obj.total ?? pagination.total ?? items.length) || 0;
  const page =
    Number(pagination.page ?? fallbackParams?.page ?? defaultPagination.page) ||
    defaultPagination.page;
  const pageSize =
    Number(
      pagination.page_size ??
        fallbackParams?.page_size ??
        defaultPagination.page_size,
    ) || defaultPagination.page_size;
  const totalPages =
    Number(
      pagination.total_pages ?? Math.ceil(total / Math.max(pageSize, 1)),
    ) || 0;

  return {
    items,
    total,
    pagination: {
      page,
      page_size: pageSize,
      total,
      total_pages: totalPages,
      has_prev: Boolean(pagination.has_prev ?? page > 1),
      has_next: Boolean(
        pagination.has_next ?? (totalPages > 0 && page < totalPages),
      ),
    },
    sort: {
      sort_by: String(
        sort.sort_by ?? fallbackParams?.sort_by ?? "direction_id",
      ) as DirectionSortBy,
      sort_dir: String(
        sort.sort_dir ?? fallbackParams?.sort_dir ?? "asc",
      ) as DirectionSortDir,
    },
  };
}

function compactParams(
  params: FetchDirectionsParams,
): Record<string, string | number> {
  const result: Record<string, string | number> = {};

  Object.entries(params).forEach(([key, value]) => {
    if (
      value === undefined ||
      value === null ||
      value === "" ||
      value === "all"
    )
      return;
    result[key] = value;
  });

  return result;
}

export async function fetchDirections(
  params: FetchDirectionsParams = {},
): Promise<DirectionListResponse> {
  const { data } = await api.get<unknown>("/admin/directions", {
    params: compactParams(params),
  });
  return normalizeDirectionListResponse(data, params);
}

export type UpdateDirectionPayload = {
  category_id?: number | null;
  cefr_level?: string | null;
  source_lang_code?: string;
  target_lang_code?: string;
  source_value?: string;
  target_value?: string;
  importance_score?: number;
  concept_base_difficulty?: number;
  source_form_score?: number;
  target_form_score?: number;
  direction_bias?: number;
  synonym_relief?: number;
  final_difficulty?: number | null;
  is_active?: boolean;
};

export async function updateDirection(
  id: number,
  payload: UpdateDirectionPayload,
): Promise<TrainingDirection> {
  const { data } = await api.patch<unknown>(`/admin/directions/${id}`, payload);
  return normalizeDirection(data);
}


export type UpdateCategoryDirectionsPayload = {
  direction_ids: number[];
  mode?: "replace" | "add" | "remove";
};

export type UpdateCategoryDirectionsResponse = {
  category_id: number;
  mode: string;
  direction_count: number;
  representative_count?: number;
  concept_count?: number;
  affected_concepts?: number;
  affected_directions?: number;
  associated_direction_count?: number;
  snapshot_version?: number;
};

export type CategoryDirectionSelectionResponse = {
  category_id: number;
  items: TrainingDirection[];
  direction_ids: number[];
  concept_ids: number[];
  representative_count: number;
  associated_direction_count: number;
  has_explicit_representatives: boolean;
};

export async function fetchCategoryDirections(
  categoryId: number,
  params: FetchDirectionsParams = {},
): Promise<DirectionListResponse> {
  const { data } = await api.get<unknown>(
    `/admin/categories/${categoryId}/directions`,
    { params: compactParams(params) },
  );
  return normalizeDirectionListResponse(data, params);
}


export type FetchCategoryDirectionSelectionParams = {
  sort_by?: DirectionSortBy;
  sort_dir?: DirectionSortDir;
};

export async function fetchCategoryDirectionSelection(
  categoryId: number,
  params: FetchCategoryDirectionSelectionParams = {},
): Promise<CategoryDirectionSelectionResponse> {
  const { data } = await api.get<unknown>(
    `/admin/categories/${categoryId}/direction-selection`,
    { params: compactParams(params) },
  );

  const obj = data && typeof data === "object" ? (data as Record<string, unknown>) : {};
  const items = directionItemsFromResponse(obj).map(normalizeDirection);

  return {
    category_id: Number(obj.category_id ?? categoryId) || categoryId,
    items,
    direction_ids: Array.isArray(obj.direction_ids)
      ? obj.direction_ids.map((value) => Number(value)).filter((value) => value > 0)
      : items.map((item) => item.direction_id),
    concept_ids: Array.isArray(obj.concept_ids)
      ? obj.concept_ids.map((value) => Number(value)).filter((value) => value > 0)
      : Array.from(new Set(items.map((item) => item.concept_id))),
    representative_count: Number(obj.representative_count ?? items.length) || items.length,
    associated_direction_count: Number(obj.associated_direction_count ?? items.length) || items.length,
    has_explicit_representatives: Boolean(obj.has_explicit_representatives),
  };
}

export async function updateCategoryDirections(
  categoryId: number,
  payload: UpdateCategoryDirectionsPayload,
): Promise<UpdateCategoryDirectionsResponse> {
  const { data } = await api.post<UpdateCategoryDirectionsResponse>(
    `/admin/categories/${categoryId}/directions`,
    {
      direction_ids: Array.from(new Set(payload.direction_ids)).filter((id) => id > 0),
      mode: payload.mode ?? "replace",
    },
  );
  return data;
}
