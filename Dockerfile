FROM scratch

COPY ditm .

EXPOSE 80

CMD ["./ditm"]
