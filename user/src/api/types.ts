export interface UserProfileData {
    id: number
    email: string
    nickname: string
    email_verified_at?: string | null
    locale: string
    member_level_id?: number
    total_recharged?: number | string
    total_spent?: number | string
    email_change_mode?: 'bind_only' | 'change_with_old_and_new'
    password_change_mode?: 'set_without_old' | 'change_with_old'
}

export interface PublicMemberLevel {
    id: number
    name: Record<string, string>
    slug: string
    icon: string
    discount_rate: number
    recharge_threshold: number
    spend_threshold: number
    is_default: boolean
    sort_order: number
}

export interface UpdateUserProfilePayload {
    nickname?: string
    locale?: string
}

export interface UserLoginLogItem {
    id: number
    user_id: number
    email: string
    status: string
    fail_reason?: string
    client_ip?: string
    user_agent?: string
    login_source?: string
    created_at?: string
}

export interface SendChangeEmailCodePayload {
    kind: 'old' | 'new'
    new_email?: string
}

export interface ChangeEmailPayload {
    new_email: string
    old_code?: string
    new_code: string
}

export interface ChangeUserPasswordPayload {
    old_password?: string
    new_password: string
}

export interface TelegramAuthPayload {
    id: number
    first_name?: string
    last_name?: string
    username?: string
    photo_url?: string
    auth_date: number
    hash: string
}

export interface TelegramMiniAppAuthPayload {
    init_data: string
}

export interface TelegramBindingData {
    bound: boolean
    provider?: string
    provider_user_id?: string
    username?: string
    avatar_url?: string
    auth_at?: string | null
    updated_at?: string | null
}

export interface WalletAccountData {
    balance: string
}

export interface WalletTransactionData {
    id: number
    type: string
    direction: string
    amount: string
    balance_after: string
    remark: string
    created_at: string
}

export interface WalletRechargePayload {
    amount: string
    channel_id: number
    currency?: string
    remark?: string
}

export interface WalletRechargeOrderData {
    id: number
    recharge_no: string
    amount: string
    payable_amount: string
    fee_amount: string
    currency: string
    status: string
    remark: string
    paid_at?: string
    created_at: string
}

export interface WalletRechargeResult {
    recharge?: WalletRechargeOrderData
    recharge_no?: string
    recharge_status?: string
    account?: WalletAccountData
    payment_id?: number
    provider_type?: string
    channel_type?: string
    interaction_mode?: string
    pay_url?: string
    qr_code?: string
    expires_at?: string
    status?: string
}

export interface GiftCardData {
    id: number
    name: string
    code: string
    amount: string
    currency: string
    status: string
    redeemed_at?: string
}

export interface GiftCardRedeemResult {
    gift_card: GiftCardData
    wallet: WalletAccountData
    transaction: WalletTransactionData
    wallet_delta: string
}

export interface AffiliateDashboardData {
    opened: boolean
    affiliate_code: string
    promotion_path: string
    click_count: number
    valid_order_count: number
    conversion_rate: number
    pending_commission: string
    available_commission: string
    withdrawn_commission: string
}

export interface AffiliateCommissionData {
    id: number
    commission_type: string
    commission_amount: string
    status: string
    confirm_at?: string
    available_at?: string
    created_at: string
}

export interface AffiliateWithdrawData {
    id: number
    amount: string
    channel: string
    account: string
    status: string
    reject_reason?: string
    created_at: string
}

export interface AffiliateWithdrawApplyPayload {
    amount: string
    channel: string
    account: string
}

export interface CreatePaymentPayload {
    order_no: string
    channel_id?: number
    use_balance?: boolean
}

export interface PaymentCreateResult {
    order_paid?: boolean
    wallet_paid_amount?: string
    online_pay_amount?: string
    payment_id?: number
    order_no?: string
    channel_id?: number
    provider_type?: string
    channel_type?: string
    interaction_mode?: string
    pay_url?: string
    qr_code?: string
    expires_at?: string
}

export interface CaptchaPayload {
    captcha_id?: string
    captcha_code?: string
    turnstile_token?: string
}
