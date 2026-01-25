// ilk önce compose aç.
    cd ~/okey101-server/infra         // önce bu klasöre git çünkü compose burada kurulu.
    docker compose up         // sonra compose ayağa kaldır.... ve bu terminali kapatma
        docker compose up -d      // eğer terminal açık kalmasını istemiyorsan bunu yap. arka planda çalışır.
        docker compose logs -f api       // compose loglarını görmek için eğer arka planda çalıştırmak istiyorsan.

// eskiyi silip yeniden başlatmak için kullan 
    docker compose down -v
    docker compose up --build


    // Kontorl için bunları kullan. compose da eğer  API listening on :8080 gelmezse alttakileri uygula....
        cd ~/okey101-server/infra       // önce bunu çalıştır.
        docker compose logs -f api      // sonra bunu cevap olarak şu gelmesi lazım...   2026/01/06 22:16:09 API listening on :8080
        curl http://localhost:8080/health  // Cevap ok. gelmesi lazım....

    cd ~/okey101-server/infra           // restart için gerekli
    docker compose restart api         // Eğer kodları değiştirirsen bunu yapki compose algılasın...

    // Eğer docker-compose.yml değişirse aşağıdakileri yap.
        docker compose down
        docker compose up


// 2. işlem test aşaması yeni bir kullanıcı için yeni terminal şunu yap.

    docker run --rm -it --network infra_default nicolaka/netshoot websocat ws://okey101-api:8080/ws     // test için container bunu kullan.

        docker exec -it okey101-api sh -c 'cd /app && go run -tags solvertest solvertest.go solver.go'  //solver.go test için

{"t":"GAME_START","reqId":"3","p":{"roomId":"A3BNV3","userId":"u1"}}


{"t":"HELLO","p":{"userId":"u1"}}
{"t":"ROOM_CREATE"}


{"t":"HELLO","p":{"userId":"u4"}}
{"t":"ROOM_JOIN","p":{"roomId":"Z2CU9T"}}



