import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Auction, CreateAuctionRequest, PlaceBidRequest, PlaceBidResponse } from './auction.model';

const API = 'http://localhost:8080/api';

@Injectable({ providedIn: 'root' })
export class AuctionService {
  private http = inject(HttpClient);

  list(): Observable<Auction[]> {
    return this.http.get<Auction[]>(`${API}/auctions`);
  }

  get(id: string): Observable<Auction> {
    return this.http.get<Auction>(`${API}/auctions/${id}`);
  }

  create(req: CreateAuctionRequest): Observable<Auction> {
    return this.http.post<Auction>(`${API}/auctions`, req);
  }

  placeBid(auctionId: string, req: PlaceBidRequest): Observable<PlaceBidResponse> {
    return this.http.post<PlaceBidResponse>(`${API}/auctions/${auctionId}/bids`, req);
  }
}
