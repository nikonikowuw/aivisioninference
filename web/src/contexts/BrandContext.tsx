import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { brandConfigApi, type BrandConfig } from 'services/api';

const defaultBrandConfig: BrandConfig = {
  id: '',
  system_name: 'Niko Admin',
  logo_url: '/favicon.ico',
  created_at: '',
  updated_at: '',
};

interface BrandContextType {
  brand: BrandConfig;
  refreshBrand: () => Promise<void>;
  setBrand: (brand: BrandConfig) => void;
}

const BrandContext = createContext<BrandContextType>({
  brand: defaultBrandConfig,
  refreshBrand: async () => { },
  setBrand: () => { },
});

export function BrandProvider({ children }: { children: ReactNode }) {
  const [brand, setBrand] = useState<BrandConfig>(defaultBrandConfig);

  const refreshBrand = async () => {
    try {
      const config = await brandConfigApi.get();
      setBrand(config);
    } catch {
      setBrand(defaultBrandConfig);
    }
  };

  useEffect(() => {
    refreshBrand();
  }, []);

  return (
    <BrandContext.Provider value={{ brand, refreshBrand, setBrand }}>
      {children}
    </BrandContext.Provider>
  );
}

export const useBrand = () => useContext(BrandContext);
