export type WorkflowDocument = {
  id: string;
  documentNo: string;
  status: string;
  creatorId: string;
  version: number;
};

export type CommandResult = WorkflowDocument | null;
