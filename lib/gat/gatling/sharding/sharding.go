package sharding

const PARTITION_HASH_SEED = 0x7A5B22367996DCFD

type ShardFunc func(int64) int

type Sharder struct {
	shards int
	fn     ShardFunc
}

func NewSharder(shards int, fn ShardFunc) *Sharder {
	return &Sharder{
		shards: shards,
		fn:     fn,
	}
}

//TODO: implement hash functions
//
//  fn pg_bigint_hash(&self, key: i64) -> usize {
//      let mut lohalf = key as u32;
//      let hihalf = (key >> 32) as u32;
//      lohalf ^= if key >= 0 { hihalf } else { !hihalf };
//      Self::combine(0, Self::pg_u32_hash(lohalf)) as usize % self.shards
//  }

//  /// Example of a hashing function based on SHA1.
//  fn sha1(&self, key: i64) -> usize {
//      let mut hasher = Sha1::new();

//      hasher.update(&key.to_string().as_bytes());

//      let result = hasher.finalize();

//      // Convert the SHA1 hash into hex so we can parse it as a large integer.
//      let hex = format!("{:x}", result);

//      // Parse the last 8 bytes as an integer (8 bytes = bigint).
//      let key = i64::from_str_radix(&hex[hex.len() - 8..], 16).unwrap() as usize;

//      key % self.shards
//  }

//  #[inline]
//  fn rot(x: u32, k: u32) -> u32 {
//      (x << k) | (x >> (32 - k))
//  }

//  #[inline]
//  fn mix(mut a: u32, mut b: u32, mut c: u32) -> (u32, u32, u32) {
//      a = a.wrapping_sub(c);
//      a ^= Self::rot(c, 4);
//      c = c.wrapping_add(b);

//      b = b.wrapping_sub(a);
//      b ^= Self::rot(a, 6);
//      a = a.wrapping_add(c);

//      c = c.wrapping_sub(b);
//      c ^= Self::rot(b, 8);
//      b = b.wrapping_add(a);

//      a = a.wrapping_sub(c);
//      a ^= Self::rot(c, 16);
//      c = c.wrapping_add(b);

//      b = b.wrapping_sub(a);
//      b ^= Self::rot(a, 19);
//      a = a.wrapping_add(c);

//      c = c.wrapping_sub(b);
//      c ^= Self::rot(b, 4);
//      b = b.wrapping_add(a);

//      (a, b, c)
//  }

//  #[inline]
//  fn _final(mut a: u32, mut b: u32, mut c: u32) -> (u32, u32, u32) {
//      c ^= b;
//      c = c.wrapping_sub(Self::rot(b, 14));
//      a ^= c;
//      a = a.wrapping_sub(Self::rot(c, 11));
//      b ^= a;
//      b = b.wrapping_sub(Self::rot(a, 25));
//      c ^= b;
//      c = c.wrapping_sub(Self::rot(b, 16));
//      a ^= c;
//      a = a.wrapping_sub(Self::rot(c, 4));
//      b ^= a;
//      b = b.wrapping_sub(Self::rot(a, 14));
//      c ^= b;
//      c = c.wrapping_sub(Self::rot(b, 24));
//      (a, b, c)
//  }

//  #[inline]
//  fn combine(mut a: u64, b: u64) -> u64 {
//      a ^= b
//          .wrapping_add(0x49a0f4dd15e5a8e3 as u64)
//          .wrapping_add(a << 54)
//          .wrapping_add(a >> 7);
//      a
//  }

//  #[inline]
//  fn pg_u32_hash(k: u32) -> u64 {
//      let mut a: u32 = 0x9e3779b9 as u32 + std::mem::size_of::<u32>() as u32 + 3923095 as u32;
//      let mut b = a;
//      let c = a;

//      a = a.wrapping_add((PARTITION_HASH_SEED >> 32) as u32);
//      b = b.wrapping_add(PARTITION_HASH_SEED as u32);
//      let (mut a, b, c) = Self::mix(a, b, c);

//      a = a.wrapping_add(k);

//      let (_a, b, c) = Self::_final(a, b, c);

//      ((b as u64) << 32) | (c as u64)
//  }
