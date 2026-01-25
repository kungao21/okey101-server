# Okey 101 Online — İlerleme Checklisti (Oyun Odaklı)

> Kural: Her madde bitti → bir sonrakine geç.

## A) Local Geliştirme Ortamı (1–2 gün)
- [x] Windows + WSL2 (Ubuntu 22.04)
- [x] Docker (WSL içinde / Docker Engine)
- [x] Git repo
- [x] docker-compose ile servisler:
  - [x] Go WebSocket API
  - [x] Redis
  - [x] PostgreSQL
- [x] Çıktı: `docker compose up` ile localde her şey çalışıyor

---

## B) Oyun Sunucusu Temeli (2–3 gün)
- [x] WebSocket bağlantı protokolü (JSON mesaj formatı)
- [x] Room (Masa) modeli
  - [x] Masa ID
  - [x] 4 oyuncu slotu (seat 1..4)
  - [x] Oyuncu durumu (connected / disconnected)
- [x] Sıra sistemi
  - [x] Kimin sırası (turnSeat)
  - [x] Oyun fazları (turnPhase: WAIT_DRAW / WAIT_DISCARD)
- [ ] **Süre / Timeout sistemi**
  - [ ] Oyuncu hamle süresi (örn. 30 sn)
  - [ ] Süre dolunca otomatik hamle (PASS / AUTO_DISCARD)
  - [ ] Disconnect + süre dolumu davranışı
- [x] Çıktı: 2+ kişi bağlanıp sıraya göre aksiyon gönderebiliyor

> Not: Timeout, bilinçli olarak C’den önce ertelendi.
> Çünkü taş/sıra mantığı oturmadan timeout yazmak hataya açıktı.

---

## C) Okey 101 Oyun Mantığı (5–7 gün)
- [ ] Taş seti (106 taş)
- [ ] Okey belirleme
- [ ] Başlangıç dağıtımı
- [ ] Hamle kuralları:
  - [ ] Taş çekme (server doğrular)
  - [ ] Taş atma (server doğrular)
  - [ ] El açma (server doğrular)
- [ ] Sunucu tarafı doğrulama (hile engeli)
- [ ] Çıktı: Kurallara uygun oynanabilen Okey 101

---

## D) Matchmaking & Redis (2–3 gün)
- [ ] Bekleme kuyruğu
- [ ] 4 kişi olunca masa oluşturma
- [ ] Oyuncu → masa eşlemesi
- [ ] Sunucu yeniden başlasa bile state toparlama
- [ ] Çıktı: “Oyna” diyen herkes otomatik masaya düşüyor

---

## E) Unity Client Entegrasyonu (5–7 gün)
- [ ] WebSocket client
- [ ] UI:
  - [ ] Masa
  - [ ] Istaka
  - [ ] Taş sürükle-bırak
- [ ] Server mesajlarını işleme
- [ ] Disconnect / reconnect
- [ ] Çıktı: Telefonda/PC’de oynanabilir build

---

## F) Kalıcı Veri (PostgreSQL) (2–3 gün)
- [ ] Kullanıcı (guest / login)
- [ ] Maç sonucu
- [ ] Puanlar
- [ ] Ceza kayıtları
- [ ] Çıktı: Oyun geçmişi ve puanlar kaydoluyor

---

## G) Performans & Güvenlik (2–3 gün)
- [ ] Aynı anda çok masa
- [ ] Rate limit
- [ ] Basit anti-cheat
- [ ] Loglama
- [ ] Çıktı: Stabil local stres testleri

---

## H) VPS’e Taşıma (1 gün)
- [ ] VPS (Hetzner / DO vs) + Ubuntu
- [ ] Docker
- [ ] Repo clone
- [ ] `docker compose up -d`
- [ ] Çıktı: Localde ne varsa VPS’te birebir çalışıyor

---

## I) Yayın Öncesi (opsiyonel)
- [ ] Domain
- [ ] SSL
- [ ] Basit load balancer (ileride)
- [ ] Monitoring
