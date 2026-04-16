from collections import Counter
from datetime import datetime
from typing import List

from fastapi import FastAPI
from pydantic import BaseModel


class Seller(BaseModel):
    external_id: str
    name: str
    marketplace: str
    rating: float
    total_orders: int
    successful_orders: int
    complaints_count: int
    returns_count: int
    average_review_score: float


class Product(BaseModel):
    external_id: str
    seller_id: str
    title: str
    category: str
    rating: float
    review_count: int
    one_star_count: int
    five_star_count: int


class Review(BaseModel):
    external_id: str
    rating: int
    text: str
    is_verified: bool
    created_at: str


class AnalyzeRequest(BaseModel):
    seller: Seller
    product: Product
    reviews: List[Review]


class AnalyzeResponse(BaseModel):
    fake_review_risk: float
    rating_manipulation_risk: float
    text_similarity_risk: float
    review_burst_risk: float
    warnings: List[str]
    trust_penalty: float
    authenticity_score: float


app = FastAPI(title="Seller Trust ML Service", version="0.1.0")


def clamp(value: float, low: float = 0.0, high: float = 1.0) -> float:
    return max(low, min(value, high))


def parse_review_times(reviews: List[Review]) -> List[datetime]:
    parsed = []
    for review in reviews:
        try:
            parsed.append(datetime.fromisoformat(review.created_at.replace("Z", "+00:00")))
        except ValueError:
            continue
    return parsed


def calc_text_similarity_risk(reviews: List[Review]) -> float:
    if not reviews:
        return 0.0

    normalized = [" ".join(review.text.lower().split()) for review in reviews if review.text.strip()]
    if not normalized:
        return 0.0

    most_common_count = Counter(normalized).most_common(1)[0][1]
    return clamp(most_common_count / len(normalized))


def calc_unverified_five_star_risk(reviews: List[Review]) -> float:
    if not reviews:
        return 0.0

    suspicious = 0
    for review in reviews:
        if review.rating == 5 and not review.is_verified:
            suspicious += 1
    return clamp(suspicious / len(reviews))


def calc_burst_risk(reviews: List[Review]) -> float:
    timestamps = parse_review_times(reviews)
    if len(timestamps) < 3:
        return 0.0

    per_day = Counter(ts.date().isoformat() for ts in timestamps)
    max_same_day = max(per_day.values())
    return clamp(max_same_day / len(reviews))


def calc_rating_manipulation_risk(product: Product) -> float:
    if product.review_count <= 0:
        return 0.0

    five_star_ratio = product.five_star_count / product.review_count
    one_star_ratio = product.one_star_count / product.review_count
    inflated_top_score = clamp((five_star_ratio - 0.78) / 0.22)
    contradiction = clamp(one_star_ratio / 0.10) * 0.4 if product.rating > 4.7 else 0.0
    return clamp(inflated_top_score * 0.6 + contradiction)


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.post("/analyze", response_model=AnalyzeResponse)
def analyze(payload: AnalyzeRequest) -> AnalyzeResponse:
    warnings: List[str] = []

    text_similarity_risk = calc_text_similarity_risk(payload.reviews)
    fake_review_risk = calc_unverified_five_star_risk(payload.reviews)
    review_burst_risk = calc_burst_risk(payload.reviews)
    rating_manipulation_risk = calc_rating_manipulation_risk(payload.product)

    if text_similarity_risk > 0.60:
        warnings.append("Высокая повторяемость текста отзывов")
    if fake_review_risk > 0.45:
        warnings.append("Слишком много непроверенных пятизвездочных отзывов")
    if review_burst_risk > 0.55:
        warnings.append("Подозрительный всплеск отзывов за короткий период")
    if rating_manipulation_risk > 0.55:
        warnings.append("Рейтинг карточки товара может быть искусственно завышен")

    trust_penalty = round(
        (
            fake_review_risk * 30
            + rating_manipulation_risk * 30
            + text_similarity_risk * 20
            + review_burst_risk * 20
        ),
        2,
    )
    authenticity_score = round(100 - trust_penalty, 2)

    return AnalyzeResponse(
        fake_review_risk=round(fake_review_risk, 3),
        rating_manipulation_risk=round(rating_manipulation_risk, 3),
        text_similarity_risk=round(text_similarity_risk, 3),
        review_burst_risk=round(review_burst_risk, 3),
        warnings=warnings,
        trust_penalty=trust_penalty,
        authenticity_score=max(0.0, authenticity_score),
    )
