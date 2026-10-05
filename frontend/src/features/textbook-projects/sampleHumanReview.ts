import type { SampleHumanReviewContract } from "@/services/textbook";

export const sampleManualReviewDimensions = [
  "句子与段落负担",
  "标题承诺",
  "重复",
  "语气",
  "例子相关性",
  "章节节奏",
  "图示机会",
  "练习梯度",
] as const;

export const fallbackSampleHumanReviewContract: SampleHumanReviewContract = {
  contract_version: "inkwords.sample-human-review.v1",
  dimensions: [...sampleManualReviewDimensions],
  minimum_score: 3,
  maximum_score: 4,
  note_min_runes: 8,
  note_max_runes: 2000,
};
