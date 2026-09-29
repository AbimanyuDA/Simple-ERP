export type WorkflowDocument = {
  id: string;
  documentNo: string;
  status: string;
  creatorId: string;
  purpose?: string;
  version: number;
};

export type FulfillmentLine = {
  id: string;
  productId: string;
  quantity: number;
  unitPrice: number;
  processedQuantity: number;
  remainingQuantity: number;
};

export type WorkflowDocumentDetail = WorkflowDocument & {
  confirmed?: boolean;
  lines: FulfillmentLine[];
};

export type CommandResult = WorkflowDocument | null;
