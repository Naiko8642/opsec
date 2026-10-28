TARGET = opsec
AS = as
LD = ld

all: $(TARGET)

$(TARGET): opsec.s
	$(AS) opsec.s -o opsec.o
	$(LD) -s opsec.o -o $(TARGET)

clean:
	rm -f $(TARGET) opsec.o

.PHONY: all clean
