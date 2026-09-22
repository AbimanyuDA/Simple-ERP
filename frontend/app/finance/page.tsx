import { ModulePage } from "@/components/module-page";
import { FinanceWorkspace } from "@/features/finance/finance-workspace";

export default function Page() {
  return <ModulePage eyebrow="AP / AR" title="Finance" description="Pantau invoice dan pembayaran tanpa kompleksitas general ledger."><FinanceWorkspace /></ModulePage>;
}
