export interface Auction {
  id: string;
  sellerId: string;
  title: string;
  description: string;
  category: string;
  startPrice: number;
  minIncrement: number;
  cap: number | null; // null = Type A; set = Type B (buy-it-now cap)
  endsAt: string; // ISO-8601 UTC
  status: string;
  createdAt: string;
  highestBidAmount: number | null; // null until the first accepted bid
  highestBidderId: string | null;
}

export interface PlaceBidRequest {
  bidderId: string;
  amount: number;
}

export interface Bid {
  id: string;
  auctionId: string;
  bidderId: string;
  amount: number;
  createdAt: string;
}

export interface PlaceBidResponse {
  bid: Bid;
  auction: Auction;
}

export interface CreateAuctionRequest {
  sellerId: string;
  title: string;
  description: string;
  category: string;
  startPrice: number;
  minIncrement: number;
  cap: number | null;
  endsAt: string;
}
