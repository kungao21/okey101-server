// ======================================================
// OKEY 101 – ANA CHECKLIST (KİLİTLİ KURALLAR)
// ======================================================
// Bu checklist PDF’lerdeki 1–36 tüm dağıtım ihtimallerini,
// zar + masa dönüşü + 21/22 taş sistemini birebir uygular.
// Kurallar KESİNLİKLE DEĞİŞTİRİLEMEZ.
// ======================================================


/* ======================================================
   A) TEMEL ODA & OTURUM
   ====================================================== */

[x] WebSocket bağlantısı
[x] HELLO → HELLO_OK
[x] ROOM_CREATE (masa oluştur)
[x] ROOM_JOIN (masaya katıl)
[x] ROOM_SNAPSHOT (anlık durum)
[ ] ROOM_LEAVE (istekli ayrılma)
[ ] DISCONNECT (ws kopması)
[ ] RECONNECT (aynı userId → aynı koltuk)


/* ======================================================
   B) LOBİ & OTOMATİK BAŞLAMA
   ====================================================== */

[x] Maksimum oyuncu = 4
[x] Oda durumu: LOBBY | PLAYING
[x] Oyuncu sayısı takibi

[x] Eğer oyuncuSayısı == 4:
    [x] 5 saniyelik geri sayım başlat

[x] Geri sayım sırasında biri çıkarsa:
    [x] geri sayımı iptal et

[x] Geri sayım 0 olunca:
    [x] oyun OTOMATİK başlar
    [x] GAME_START client’tan GÖNDERİLMEZ

[x] Oyunda owner yetkisi YOK


/* ======================================================
   C) OYUN AYARLARI (MASA OLUŞURKEN)
   ====================================================== */

[ ] OyunModu:
    [ ] KLASIK_101
    [ ] KATLAMALI_101

[ ] TakımModu:
    [ ] TEKLI
    [ ] TAKIM

[ ] CezaModu:
    [ ] AÇIK
    [ ] KAPALI

[ ] ElSayısı: 1–11 arası seçilebilir

[ ] Oyun başladıktan sonra ayarlar KİLİTLENİR


/* ======================================================
   D) DAĞITAN (DEALER) & EL AKIŞI
   ====================================================== */

[ ] dealerSeat (1–4)
[ ] İlk dealer = masaya ilk oturan
[ ] Her el sonunda dealer +1 döner

[ ] dealerSeat şunları belirler:
    [ ] zar referansı
    [ ] 22 taş alan oyuncu


/* ======================================================
   E) TAŞ SETİ & DESTE OLUŞTURMA
   ====================================================== */

[x] 106 taş üretimi:
    [x] 4 renk
    [x] 1–13 sayılar
    [x] her taştan 2 adet
    [x] 2 sahte okey

[x] Taşlar karıştırılır

[ ] 15 deste oluştur:
    [ ] 14 adet 7’li deste
    [ ] 1 adet fazla taş
    [ ] deste numaraları 1–15


/* ======================================================
   F) MASA DÖNÜŞÜ (KRİTİK)
   ====================================================== */

[ ] Her el başında masa yeniden hesaplanır:
    [ ] dealer → 1–4. desteler
    [ ] sıradaki → 5–8
    [ ] sıradaki → 9–12
    [ ] son oyuncu → 13–15

[ ] Deste numaraları SABİT
[ ] Oyuncu–deste eşleşmesi döner


/* ======================================================
   G) ZAR / BAŞLANGIÇ DESTESİ / OKEY
   ====================================================== */

[ ] Zar atılır (1–6)

[ ] BaşlangıçDestesi = zar sonucu (dealer bazlı)

[ ] BaşlangıçDestesi:
    [ ] 8 taşlı deste olur

[ ] GöstergeDestesi = Başlangıç - 3
    [ ] 1–15 arası sarma

[ ] Gösterge taşı açılır

[ ] Okey:
    [ ] göstergenin +1 sayısı
    [ ] 13 → 1 sarar


/* ======================================================
   H) DAĞITIM (KİLİTLİ)
   ====================================================== */

[ ] Dağıtım başlangıcı = BaşlangıçDestesi

[ ] Tam 12 deste dağıtılır:
    [ ] saat yönünde
    [ ] deste deste

[ ] Her oyuncu:
    [ ] tam 3 deste alır
    [ ] toplam 21 taş

[ ] Dealer’ın bir sonraki oyuncusu:
    [ ] 8 taşlı desteyi alır
    [ ] toplam 22 taş

[ ] Kalan 3 deste:
    [ ] çekme destesi olur
    [ ] sıra korunur


/* ======================================================
   I) SIRA & ZAMANLAYICI
   ====================================================== */

[x] turnSeat takibi
[x] turnPhase:
    [x] TAŞ_ÇEK
    [x] TAŞ_AT

[x] Her sıra için 30 saniye

[ ] El açılırsa:
    [ ] +30 saniye bonus

[x] Süre dolarsa:
    [ ] en küçük taş otomatik atılır
    [ ] sıra ilerler


/* ======================================================
   J) EL AÇMA (101 KURALI)
   ====================================================== */

[ ] EL_AC isteği

[ ] Doğrulama:
    [ ] Klasik: toplam >= 101
    [ ] Katlamalı: önceki açmadan büyük

[ ] Açılan taşlar:
    [ ] sadece oyuncunun alanında
    [ ] ortak havuz YOK


/* ======================================================
   K) HATALI EL AÇMA & CEZA
   ====================================================== */

[ ] El açma geçersizse:
    [ ] taşlar geri alınır
    [ ] 101 ceza yazılır
    [ ] otomatik taş atılır

[ ] Tekrar el açmaya izin verilir
[ ] Her hatada +101 ceza eklenir


/* ======================================================
   L) İŞLEK (AÇILAN ELE EKLEME)
   ====================================================== */

[ ] Sadece el açan oyuncu işlek yapabilir

[ ] Otomatik işlek
[ ] Manuel işlek

[ ] İşlek server tarafından doğrulanır

[ ] Hatalı işlek:
    [ ] taş geri verilir


/* ======================================================
   M) ÇEKME DESTESİ
   ====================================================== */

[ ] Taşlar üstten çekilir

[ ] Çekme destesi biterse:
    [ ] el biter


/* ======================================================
   N) EL BİTİŞİ & PUAN
   ====================================================== */

[ ] El bitişi tespiti

[ ] El içi cezalar uygulanır

[ ] El kazananı belirlenir


/* ======================================================
   O) OYUN BİTİŞİ & SIRALAMA
   ====================================================== */

[ ] N el sonunda:
    [ ] genel sıralama yapılır

[ ] Puanlama:
    [ ] 1. → +n
    [ ] 2. → +(n-2)
    [ ] 3. → 0
    [ ] 4. → -4

[ ] Oyuncu puanları kaydedilir


/* ======================================================
   P) OTOMATİK YENİDEN BAŞLAMA
   ====================================================== */

[ ] Oyun bitince:
    [ ] 4 kişi varsa → 5 sn geri sayım
    [ ] biri çıkarsa → iptal

[ ] Aynı ayarlarla yeni oyun


/* ======================================================
   Q) OYUNCU YARDIMLARI (OPSİYONEL)
   ====================================================== */

[ ] SERİ_ÖNER
[ ] ÇİFT_ÖNER
[ ] SERVER_HESAPLI_EN_İYİ_DİZİLİM


/* ======================================================
   R) GELİŞTİRME & TEST
   ====================================================== */

[x] Websocat testleri
[x] Sıra zaman aşımı testleri
[ ] Zar dağılım testleri
[ ] PDF 36 senaryo birebir doğrulama
