#!/usr/bin/env python3
"""Generate synthetic bank retail-marketing datasets for the StarData dev project.

Outputs gzip CSV files to dev-project/data/:
  dim_customer (50k), dim_product (~40), dim_campaign (24),
  fact_marketing_touch (500k), fact_transaction (1M), fact_aum_snapshot (600k)

Deterministic (seeded). Pure stdlib, no third-party deps.
"""

import csv
import gzip
import os
import random
from datetime import date, datetime, timedelta

OUT_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "dev-project", "data")
rng = random.Random(42)

N_CUSTOMERS = 50_000
N_TOUCHES = 500_000
N_TXNS = 1_000_000
DATA_START = date(2024, 8, 1)
DATA_END = date(2026, 7, 26)
SNAPSHOT_MONTHS = [date(2025, m, 1) for m in range(8, 13)] + [date(2026, m, 1) for m in range(1, 8)]

SURNAMES = "王李张刘陈杨黄赵吴周徐孙马朱胡郭何林罗高郑梁谢宋唐许韩冯邓曹彭曾肖田董潘袁蔡蒋余于杜叶程苏魏吕丁任沈姚卢姜崔钟谭陆汪范金石廖贾夏韦付方白邹孟熊秦邱江尹薛闫段雷侯龙史陶黎贺顾毛郝龚邵万钱严覃武戴莫孔向汤"
GIVEN = ("伟芳娜秀英敏静丽强磊军洋勇艳杰娟涛明超秀兰霞平刚桂英建华文辉力明永健世广志义兴良海山仁波宁贵福生龙元全国胜学祥才发武新利清飞彬富顺信子杰涛昌成康星光天达安岩中茂进林有坚和彪博诚先敬震振壮会思群豪心邦承乐绍功松善厚庆磊民友裕河哲江超浩亮政谦亨奇固之轮翰朗伯宏言若鸣朋斌梁栋维启克伦翔旭鹏泽晨辰士以建家致树炎德行时泰盛雄琛钧冠策腾楠榕风航弘")

CITIES = [("北京", 12), ("上海", 12), ("广州", 9), ("深圳", 10), ("杭州", 8), ("成都", 8),
          ("南京", 7), ("武汉", 7), ("西安", 6), ("重庆", 6), ("苏州", 6), ("天津", 5),
          ("长沙", 4), ("青岛", 4), ("郑州", 3), ("宁波", 3)]
OCCUPATIONS = [("企业职员", 30), ("工程师", 12), ("个体经营", 12), ("公务员", 8), ("教师", 7),
               ("医生", 5), ("金融从业者", 6), ("自由职业", 8), ("退休", 9), ("其他", 3)]
TIERS = [("大众", 70), ("金卡", 20), ("白金", 8), ("私行", 2)]
RISK_LEVELS = [("保守型", 20), ("稳健型", 35), ("平衡型", 25), ("成长型", 14), ("进取型", 6)]
GENDERS = [("男", 51), ("女", 49)]

# AUM lognormal-ish base by tier (CNY)
TIER_AUM = {"大众": (30_000, 1.2), "金卡": (300_000, 0.9), "白金": (1_500_000, 0.7), "私行": (10_000_000, 0.6)}
# activity multiplier for transactions
TIER_ACTIVITY = {"大众": 1.0, "金卡": 2.0, "白金": 3.5, "私行": 6.0}
# marketing conversion multiplier
TIER_CONV = {"大众": 1.0, "金卡": 1.5, "白金": 2.2, "私行": 3.0}


def wchoice(pairs):
    return rng.choices([p[0] for p in pairs], weights=[p[1] for p in pairs], k=1)[0]


def writer(name):
    os.makedirs(OUT_DIR, exist_ok=True)
    f = gzip.open(os.path.join(OUT_DIR, name + ".csv.gz"), "wt", newline="", encoding="utf-8")
    return f, csv.writer(f)


def gen_customers():
    f, w = writer("dim_customer")
    w.writerow(["customer_id", "customer_name", "gender", "age", "occupation", "city",
                "customer_tier", "risk_level", "is_payroll", "open_date", "aum_base"])
    customers = []
    for i in range(1, N_CUSTOMERS + 1):
        cid = f"C{i:08d}"
        name = rng.choice(SURNAMES) + "".join(rng.sample(GIVEN, rng.choice([1, 2])))
        gender = wchoice(GENDERS)
        age = min(80, max(20, int(rng.gauss(42, 13))))
        occupation = "退休" if age >= 62 and rng.random() < 0.8 else wchoice(OCCUPATIONS)
        city = wchoice(CITIES)
        tier = wchoice(TIERS)
        risk = wchoice(RISK_LEVELS)
        payroll = rng.random() < (0.45 if occupation in ("企业职员", "公务员", "教师", "医生", "工程师") else 0.15)
        open_date = DATA_START - timedelta(days=rng.randint(30, 365 * 12))
        mu, sigma = TIER_AUM[tier]
        aum = round(mu * rng.lognormvariate(0, sigma), 2)
        w.writerow([cid, name, gender, age, occupation, city, tier, risk,
                    "true" if payroll else "false", open_date.isoformat(), aum])
        customers.append((cid, tier, aum))
    f.close()
    return customers


PRODUCTS = [
    # (id, name, type, risk, expected_return %)
    ("P001", "活期存款", "存款", "R1", 0.25), ("P002", "定期存款1年期", "存款", "R1", 1.65),
    ("P003", "定期存款3年期", "存款", "R1", 2.15), ("P004", "大额存单2年期", "存款", "R1", 2.30),
    ("P005", "大额存单3年期", "存款", "R1", 2.55), ("P006", "结构性存款90天", "存款", "R1", 2.80),
    ("P101", "稳健理财30天", "理财", "R2", 2.60), ("P102", "稳健理财90天", "理财", "R2", 2.95),
    ("P103", "稳健理财180天", "理财", "R2", 3.20), ("P104", "增利理财365天", "理财", "R3", 3.60),
    ("P105", "尊享理财私行专属", "理财", "R3", 4.10), ("P106", "现金管理类理财", "理财", "R1", 2.10),
    ("P201", "货币市场基金A", "基金", "R1", 1.90), ("P202", "纯债基金C", "基金", "R2", 3.40),
    ("P203", "二级债基A", "基金", "R3", 4.50), ("P204", "沪深300指数基金", "基金", "R4", 6.50),
    ("P205", "科技主题混合基金", "基金", "R5", 9.00), ("P206", "消费主题股票基金", "基金", "R5", 8.00),
    ("P207", "养老目标基金2045", "基金", "R3", 5.00), ("P208", "黄金ETF联接基金", "基金", "R4", 5.50),
    ("P301", "终身寿险", "保险", "R1", 2.50), ("P302", "年金保险尊享版", "保险", "R1", 2.80),
    ("P303", "重大疾病保险", "保险", "R1", 0.00), ("P304", "增额终身寿3.0", "保险", "R1", 2.90),
    ("P401", "个人住房按揭贷款", "贷款", "R1", 3.45), ("P402", "个人消费贷款", "贷款", "R2", 3.80),
    ("P403", "个人经营贷款", "贷款", "R2", 3.60), ("P404", "汽车分期贷款", "贷款", "R2", 4.20),
    ("P501", "白金信用卡", "信用卡", "R1", 0.00), ("P502", "标准金卡", "信用卡", "R1", 0.00),
    ("P503", "车主信用卡", "信用卡", "R1", 0.00), ("P504", "航空联名卡", "信用卡", "R1", 0.00),
]


def gen_products():
    f, w = writer("dim_product")
    w.writerow(["product_id", "product_name", "product_type", "risk_level", "expected_return_pct"])
    for row in PRODUCTS:
        w.writerow(row)
    f.close()


CHANNELS = [("APP推送", 30), ("短信", 28), ("企业微信", 16), ("外呼", 14), ("网点", 12)]
# channel -> (delivery rate, click/pickup rate)
CHANNEL_FUNNEL = {"APP推送": (0.97, 0.16), "短信": (0.99, 0.05), "企业微信": (0.95, 0.14),
                  "外呼": (0.82, 0.28), "网点": (1.00, 0.38)}
THEMES = [("新客获客", "存款"), ("存量激活", "存款"), ("理财转化", "理财"), ("基金定投推广", "基金"),
          ("信用卡分期营销", "信用卡"), ("贵宾客户升级", "理财"), ("保险保障计划", "保险"), ("消费贷促动", "贷款")]

CAMPAIGN_NAMES = [
    ("MKT2024001", "2024金秋存款季", "存量激活", date(2024, 9, 1), date(2024, 10, 15)),
    ("MKT2024002", "新客见面礼", "新客获客", date(2024, 8, 15), date(2024, 9, 30)),
    ("MKT2024003", "双十一信用卡分期节", "信用卡分期营销", date(2024, 10, 25), date(2024, 11, 15)),
    ("MKT2024004", "年末理财冲刺", "理财转化", date(2024, 11, 20), date(2024, 12, 31)),
    ("MKT2024005", "养老保障月", "保险保障计划", date(2024, 10, 1), date(2024, 10, 31)),
    ("MKT2025001", "2025开门红存款营销", "存量激活", date(2025, 1, 1), date(2025, 2, 28)),
    ("MKT2025002", "春节消费贷礼包", "消费贷促动", date(2025, 1, 10), date(2025, 2, 10)),
    ("MKT2025003", "白金客户理财升级", "贵宾客户升级", date(2025, 3, 1), date(2025, 4, 15)),
    ("MKT2025004", "基金定投启航计划", "基金定投推广", date(2025, 3, 15), date(2025, 5, 15)),
    ("MKT2025005", "五一新客大礼包", "新客获客", date(2025, 4, 20), date(2025, 5, 20)),
    ("MKT2025006", "618信用卡消费返现", "信用卡分期营销", date(2025, 6, 1), date(2025, 6, 30)),
    ("MKT2025007", "年中理财节", "理财转化", date(2025, 6, 15), date(2025, 7, 31)),
    ("MKT2025008", "暑期出行卡权益", "信用卡分期营销", date(2025, 7, 1), date(2025, 8, 15)),
    ("MKT2025009", "金秋保险守护季", "保险保障计划", date(2025, 9, 1), date(2025, 10, 15)),
    ("MKT2025010", "国庆消费贷特惠", "消费贷促动", date(2025, 9, 20), date(2025, 10, 20)),
    ("MKT2025011", "双十一分期0手续费", "信用卡分期营销", date(2025, 10, 25), date(2025, 11, 15)),
    ("MKT2025012", "年终奖理财规划", "理财转化", date(2025, 12, 1), date(2026, 1, 15)),
    ("MKT2025013", "私行客户答谢季", "贵宾客户升级", date(2025, 11, 1), date(2025, 12, 31)),
    ("MKT2026001", "2026开门红揽储", "存量激活", date(2026, 1, 1), date(2026, 2, 28)),
    ("MKT2026002", "春季基金定投月", "基金定投推广", date(2026, 3, 1), date(2026, 3, 31)),
    ("MKT2026003", "新客春日礼", "新客获客", date(2026, 3, 10), date(2026, 4, 30)),
    ("MKT2026004", "五一信用卡消费季", "信用卡分期营销", date(2026, 4, 25), date(2026, 5, 25)),
    ("MKT2026005", "年中财富节", "理财转化", date(2026, 6, 1), date(2026, 7, 15)),
    ("MKT2026006", "暑期消费贷清凉价", "消费贷促动", date(2026, 7, 1), date(2026, 7, 26)),
]


def gen_campaigns():
    f, w = writer("dim_campaign")
    w.writerow(["campaign_id", "campaign_name", "theme", "channel", "target_tier",
                "start_date", "end_date", "budget"])
    campaigns = []
    theme_type = dict(THEMES)
    for cid, name, theme, start, end in CAMPAIGN_NAMES:
        channel = wchoice(CHANNELS)
        target = "白金" if "白金" in name or "私行" in name or theme == "贵宾客户升级" else wchoice(
            [("全部", 55), ("大众", 15), ("金卡", 20), ("白金", 10)])
        budget = rng.choice([200_000, 300_000, 500_000, 800_000, 1_200_000, 2_000_000])
        w.writerow([cid, name, theme, channel, target, start.isoformat(), end.isoformat(), budget])
        campaigns.append((cid, theme, theme_type[theme], channel, target, start, end))
    f.close()
    return campaigns


def gen_touches(customers, campaigns):
    f, w = writer("fact_marketing_touch")
    w.writerow(["touch_id", "campaign_id", "customer_id", "touch_time", "channel",
                "is_delivered", "is_clicked", "is_converted", "convert_amount"])
    # pre-bucket customers by tier for targeting
    by_tier = {}
    for c in customers:
        by_tier.setdefault(c[1], []).append(c)
    camp_weights = [rng.uniform(0.5, 1.5) for _ in campaigns]
    for i in range(1, N_TOUCHES + 1):
        cid, theme, ptype, channel, target, start, end = rng.choices(campaigns, weights=camp_weights, k=1)[0]
        if target != "全部" and rng.random() < 0.7:
            cust = rng.choice(by_tier[target])
        else:
            cust = rng.choice(customers)
        days = (end - start).days
        t = datetime.combine(start, datetime.min.time()) + timedelta(
            days=rng.randint(0, max(days, 1)), hours=rng.randint(9, 20), minutes=rng.randint(0, 59))
        deliver_rate, click_rate = CHANNEL_FUNNEL[channel]
        delivered = rng.random() < deliver_rate
        clicked = delivered and rng.random() < click_rate
        conv_rate = 0.22 * TIER_CONV[cust[1]]
        converted = clicked and rng.random() < min(conv_rate, 0.8)
        amount = 0.0
        if converted:
            base = {"存款": 80_000, "理财": 150_000, "基金": 40_000, "保险": 25_000,
                    "贷款": 120_000, "信用卡": 6_000}[ptype]
            amount = round(base * TIER_CONV[cust[1]] * rng.lognormvariate(0, 0.8), 2)
        w.writerow([f"T{i:09d}", cid, cust[0], t.strftime("%Y-%m-%d %H:%M:%S"), channel,
                    "true" if delivered else "false", "true" if clicked else "false",
                    "true" if converted else "false", amount])
    f.close()


TXN_CHANNELS = [("手机银行", 55), ("网上银行", 12), ("柜面", 10), ("ATM", 8), ("第三方支付", 15)]
TYPE_TXN = {"存款": [("存入", 55), ("支取", 45)], "理财": [("申购", 60), ("赎回", 40)],
            "基金": [("申购", 62), ("赎回", 38)], "保险": [("申购", 100)],
            "贷款": [("放款", 30), ("还款", 70)], "信用卡": [("消费", 78), ("还款", 22)]}
TYPE_AMT = {"存款": (30_000, 1.1), "理财": (100_000, 0.9), "基金": (15_000, 1.1),
            "保险": (20_000, 0.8), "贷款": (60_000, 1.0), "信用卡": (800, 1.2)}
# monthly seasonality weight (Jan spike 开门红, Dec year-end)
MONTH_W = {1: 1.5, 2: 0.9, 3: 1.1, 4: 1.0, 5: 1.0, 6: 1.15, 7: 1.0, 8: 0.95,
           9: 1.05, 10: 1.0, 11: 1.2, 12: 1.35}


def gen_transactions(customers):
    f, w = writer("fact_transaction")
    w.writerow(["txn_id", "customer_id", "product_id", "txn_time", "txn_type",
                "txn_channel", "txn_amount"])
    cust_weights = [TIER_ACTIVITY[c[1]] for c in customers]
    prod_weights = [{"存款": 26, "理财": 18, "基金": 14, "保险": 4, "贷款": 8, "信用卡": 30}[p[2]]
                    for p in PRODUCTS]
    all_days = []
    d = DATA_START
    while d <= DATA_END:
        all_days.append(d)
        d += timedelta(days=1)
    day_weights = [MONTH_W[dd.month] * (0.75 if dd.weekday() >= 5 else 1.0) for dd in all_days]
    picked_custs = rng.choices(customers, weights=cust_weights, k=N_TXNS)
    picked_prods = rng.choices(PRODUCTS, weights=prod_weights, k=N_TXNS)
    picked_days = rng.choices(all_days, weights=day_weights, k=N_TXNS)
    for i in range(N_TXNS):
        cust = picked_custs[i]
        prod = picked_prods[i]
        dd = picked_days[i]
        t = datetime.combine(dd, datetime.min.time()) + timedelta(
            hours=rng.randint(7, 22), minutes=rng.randint(0, 59), seconds=rng.randint(0, 59))
        ttype = wchoice(TYPE_TXN[prod[2]])
        mu, sigma = TYPE_AMT[prod[2]]
        amount = round(mu * (TIER_ACTIVITY[cust[1]] ** 0.5) * rng.lognormvariate(0, sigma), 2)
        w.writerow([f"X{i + 1:09d}", cust[0], prod[0], t.strftime("%Y-%m-%d %H:%M:%S"),
                    ttype, wchoice(TXN_CHANNELS), amount])
    f.close()


def gen_snapshots(customers):
    f, w = writer("fact_aum_snapshot")
    w.writerow(["customer_id", "snapshot_month", "deposit_balance", "wealth_balance",
                "fund_balance", "aum"])
    for cust in customers:
        cid, tier, aum0 = cust
        # per-customer stable split with drift
        dep_r = rng.uniform(0.25, 0.75)
        wea_r = rng.uniform(0.1, 0.9) * (1 - dep_r)
        growth = rng.uniform(-0.004, 0.014)  # monthly drift
        for mi, month in enumerate(SNAPSHOT_MONTHS):
            level = aum0 * ((1 + growth) ** mi) * rng.uniform(0.96, 1.04)
            uplift = 1.06 if month.month == 12 else (1.04 if month.month == 1 else 1.0)
            dep = level * dep_r * uplift
            wea = level * wea_r
            fund = max(level - dep - wea, 0)
            w.writerow([cid, month.isoformat(), round(dep, 2), round(wea, 2),
                        round(fund, 2), round(dep + wea + fund, 2)])
    f.close()


def main():
    print("generating dim_customer ...")
    customers = gen_customers()
    print("generating dim_product ...")
    gen_products()
    print("generating dim_campaign ...")
    campaigns = gen_campaigns()
    print("generating fact_marketing_touch ...")
    gen_touches(customers, campaigns)
    print("generating fact_transaction ...")
    gen_transactions(customers)
    print("generating fact_aum_snapshot ...")
    gen_snapshots(customers)
    print("done ->", os.path.abspath(OUT_DIR))


if __name__ == "__main__":
    main()
