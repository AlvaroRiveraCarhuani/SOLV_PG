export interface RubricLevelDTO {
  id?: string;
  name: string;
  score: number;
  description: string;
}

export interface RubricCriterionDTO {
  id?: string;
  name: string;
  description: string;
  weight: number;
  levels: RubricLevelDTO[];
}

export interface RubricDTO {
  id?: string;
  exercise_id?: string;
  criteria: RubricCriterionDTO[];
}
