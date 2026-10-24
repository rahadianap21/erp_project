export interface PaymentMethod {
  id: number;
  code: string;
  name: string;
  channel: string;
  account_id?: number | null;
  is_cash: boolean;
  sort_order: number;
  is_active: boolean;
}

export interface POSCategory {
  id: number;
  name: string;
}

export interface Member {
  id: number;
  code: string;
  name: string;
  phone: string;
  default_term_days: number;
  points: number;
  is_active: boolean;
}

export interface Voucher {
  id: number;
  code: string;
  name: string;
  description: string;
  discount_type: string;
  discount_value: number;
  min_purchase: number;
  max_discount: number;
  valid_from?: string | null;
  valid_until?: string | null;
  is_active: boolean;
  used_at?: string | null;
}

export interface POSProduct {
  id: number;
  sku: string;
  barcode: string;
  name: string;
  selling_price: number;
  current_stock: number;
  category_name: string;
  selling_unit: string;
  discount_type: string;
  discount_value: number;
  is_taxable: boolean;
}

export interface POSBootstrap {
  payment_methods: PaymentMethod[];
  categories: POSCategory[];
  tax_rate: number;
  tax_mode: string;
}

export interface CartLine {
  product: POSProduct;
  qty: number;
}

export interface PaymentDraft {
  paymentMethodId: number;
  amount: number;
  referenceNumber: string;
}

export interface SaleDetail {
  id: number;
  product_id: number;
  sku: string;
  product_name: string;
  qty: number;
  unit_price: number;
  discount_amount: number;
  line_total: number;
}

export interface SalePayment {
  id: number;
  payment_method_id: number;
  payment_method_name: string;
  channel: string;
  amount: number;
}

export interface Sale {
  id: number;
  sale_number: string;
  sale_date: string;
  member_id: number;
  member_name: string;
  tax_type: string;
  subtotal: number;
  item_discount: number;
  discount_type: string;
  discount_value: number;
  discount_amount: number;
  dpp_amount: number;
  tax_amount: number;
  total_amount: number;
  paid_amount: number;
  change_amount: number;
  points_earned: number;
  redeem_points: number;
  redeem_amount: number;
  voucher_id?: number;
  voucher_code?: string;
  voucher_discount: number;
  payment_status: string;
  status: string;
  due_date?: string | null;
  notes: string;
  details?: SaleDetail[];
  payments?: SalePayment[];
}

export const formatRupiah = (value: number): string => {
  const rounded = Math.round(Number.isFinite(value) ? value : 0);
  const sign = rounded < 0 ? "-" : "";
  const digits = Math.abs(rounded)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  return `Rp ${sign}${digits}`;
};

export const round2 = (value: number): number => Math.round(value * 100) / 100;
