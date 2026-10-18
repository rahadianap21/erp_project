import { Loader2 } from "lucide-react";
export default function ProductsLoading() {
  return (
    <div className="w-full min-h-[60vh] flex items-center justify-center gap-2 text-slate-500 text-xs font-medium">
      <Loader2 className="size-4 animate-spin text-blue-600" />
      <span>Menghubungkan rantai pasok katalog barang dan metrika HPP...</span>
    </div>
  );
}
