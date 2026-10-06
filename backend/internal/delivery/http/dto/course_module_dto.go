package dto

type CreateModuleRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	OrderIndex  int    `json:"order_index"`
	PassScore   int    `json:"pass_score"`
}

type UpdateModuleRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	OrderIndex  int    `json:"order_index"`
	PassScore   int    `json:"pass_score"`
}

type SetPrerequisitesRequest struct {
	PrerequisiteModuleIDs []string `json:"prerequisite_module_ids"`
}

type AssignExerciseModuleRequest struct {
	ModuleID *string `json:"module_id"`
}
