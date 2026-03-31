import {createContext} from "react";
import type {SiteInfo} from "../api/endpoints";

export const SiteInfoContext = createContext<SiteInfo | null>(null);
