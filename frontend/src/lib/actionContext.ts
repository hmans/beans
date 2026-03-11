export interface ActionContext {
	onAction: (message: string) => void;
	disabled: boolean;
}
