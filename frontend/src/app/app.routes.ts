import { Routes } from '@angular/router';
import { AuctionList } from './auctions/auction-list/auction-list';
import { AuctionCreate } from './auctions/auction-create/auction-create';
import { AuctionDetail } from './auctions/auction-detail/auction-detail';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'auctions' },
  { path: 'auctions', component: AuctionList },
  { path: 'auctions/new', component: AuctionCreate },
  { path: 'auctions/:id', component: AuctionDetail },
];
